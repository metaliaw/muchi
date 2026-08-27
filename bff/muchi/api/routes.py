"""Las rutas HTTP de Muchi, pegadas al dominio sin nombrar una fuente.

Cada ruta es una traduccion: entra un request, sale un dominio, vuelve un
contrato. Los endpoints largos (refresco, indexacion, cotizacion) responden
con un flujo SSE para que la app muestre el avance, igual que hoy.
"""
from __future__ import annotations

import json
import queue
import threading
from typing import Callable

from fastapi import APIRouter, Query, Request
from fastapi.responses import Response, StreamingResponse

from muchi.mtg import history, offers as offers_core, optimizer, stores as store_index
from muchi.mtg.models import Offer
from muchi.mtg.ports import InventoryUnavailable

from .schemas import (
    CartPlanRequest,
    DecklistQuoteRequest,
    HistoryRowOut,
    InventoryStatusOut,
    InventoryListRateOut,
    OfferOut,
    OffersOut,
    RecommendationOut,
    SavePricesRequest,
    StoreStatusOut,
    offer_from_in,
    order_from_in,
)

router = APIRouter()


def sse(payload: dict) -> str:
    return f"data: {json.dumps(payload, ensure_ascii=False)}\n\n"


def _cast(request: Request):
    return request.app.state.cast


def _offer_payload(o: Offer) -> dict:
    return OfferOut.model_validate(o).model_dump()


@router.get("/suggest", response_model=list[str])
def suggest_names(request: Request, q: str = Query(min_length=2)):
    return _cast(request).primary.suggest_names(q)


@router.get("/offers", response_model=OffersOut)
def find_offers(request: Request, q: str = Query(min_length=1)):
    cast = _cast(request)
    card_id, offers = offers_core.find_offers(cast.primary, cast.extras, q)
    return {"card_id": card_id, "offers": [OfferOut.model_validate(o) for o in offers]}


@router.get("/offers/{card_id}/refresh")
def refresh_offers(request: Request, card_id: str):
    def stream():
        for p in _cast(request).primary.refresh_offers(card_id):
            yield sse({"store": p.store, "done": p.done, "total": p.total})
        yield sse({"done": True})

    return StreamingResponse(stream(), media_type="text/event-stream")


@router.get("/recommend", response_model=list[RecommendationOut])
def recommend_cards(request: Request, commander: str = Query(min_length=3)):
    return _cast(request).advisor.recommend_cards(commander)


@router.get("/history", response_model=list[HistoryRowOut])
def read_history(request: Request, card: str = Query(min_length=1)):
    return history.read_history(_cast(request).cx, card)


@router.post("/history", status_code=204)
def save_prices(request: Request, body: SavePricesRequest):
    cast = _cast(request)
    history.save_prices(cast.cx, body.card_name,
                        [offer_from_in(o) for o in body.offers])
    return Response(status_code=204)


@router.post("/decklist/quote")
def quote_decklist(request: Request, body: DecklistQuoteRequest):
    cast = _cast(request)
    orders = [order_from_in(o) for o in body.orders]

    def stream():
        total = len(orders)
        failed: list[str] = []
        for i, p in enumerate(orders):
            try:
                card_id, its_offers = offers_core.find_offers(cast.primary, cast.extras, p.name)
                if its_offers:
                    history.save_prices(cast.cx, p.name, its_offers)
                yield sse({"card": p.name, "index": i, "total": total,
                           "offers": [_offer_payload(o) for o in its_offers]})
            except Exception:
                failed.append(p.name)
                yield sse({"card": p.name, "index": i, "total": total, "error": True})
        yield sse({"done": True, "failed": failed, "total": total})

    return StreamingResponse(stream(), media_type="text/event-stream")


@router.post("/cart/plan")
def build_cart_plan(request: Request, body: CartPlanRequest):
    orders = [order_from_in(o) for o in body.orders]
    by_card = {k: [offer_from_in(o) for o in v] for k, v in body.offers_by_card.items()}
    shipping = body.shipping_per_store
    plan = (optimizer.build_optimal_plan(orders, by_card, shipping)
            if body.strategy == "optimal"
            else optimizer.build_naive_plan(orders, by_card, shipping))
    return plan


@router.get("/stores", response_model=list[StoreStatusOut])
def read_stores(request: Request):
    cast = _cast(request)
    return store_index.read_stores_status(cast.cx, cast.stores)


@router.get("/stores/blocked", response_model=dict[str, dict])
def list_blocked_stores(request: Request):
    return {s: {"url": u, "reason": r}
            for s, (u, r) in _cast(request).stores.list_blocked_stores().items()}


def _stream_blocking(fn: Callable, on_progress: Callable):
    """Corre fn(progreso) en un hilo y drena su avance como eventos SSE.

    fn recibe un callback de progreso y devuelve un dict de resultado.
    """
    q: queue.Queue = queue.Queue()

    def progress(*args):
        q.put(("p", args))

    def work():
        try:
            q.put(("done", fn(progress)))
        except Exception as e:  # noqa: BLE001
            q.put(("error", str(e)))

    threading.Thread(target=work, daemon=True).start()

    while True:
        kind, payload = q.get()
        if kind == "p":
            yield on_progress(*payload)
        elif kind == "done":
            yield {**payload, "done": True}
            return
        else:
            yield {"error": payload}
            return


@router.post("/stores/{store}/index")
def index_store(request: Request, store: str, url: str = Query()):
    cast = _cast(request)

    def run(progress):
        saved = store_index.index_store(cast.cx, cast.stores, store, url, progress)
        return {"saved": saved}

    def on_progress(products, offers):
        return {"products": products, "offers": offers}

    def stream():
        for ev in _stream_blocking(run, on_progress):
            yield sse(ev)

    return StreamingResponse(stream(), media_type="text/event-stream")


@router.post("/inventory/import")
def import_inventory(request: Request):
    cast = _cast(request)
    if cast.inventory is None:
        raise InventoryUnavailable("no hay inventario configurado")

    def run(progress):
        saved, without_price = store_index.import_inventory(cast.cx, cast.inventory, progress)
        return {"saved": saved, "without_price": without_price}

    def on_progress(done, total, label):
        return {"index": done, "total": total, "label": label}

    def stream():
        for ev in _stream_blocking(run, on_progress):
            yield sse(ev)

    return StreamingResponse(stream(), media_type="text/event-stream")


@router.get("/inventory", response_model=InventoryStatusOut | None)
def read_inventory(request: Request):
    cast = _cast(request)
    inv = cast.inventory
    if inv is None:
        return None
    status = store_index.read_inventory_status(cast.cx, inv.store)
    return {
        "store": inv.store,
        "rates": [InventoryListRateOut(label=l, rate=r) for l, r in inv.list_rates()],
        "status": StoreStatusOut.model_validate(status) if status else None,
    }
