"""La app FastAPI de Muchi: arma el elenco una vez y lo expone por HTTP.

Igual que app.py, aca entra el Cast y no se nombra una fuente concreta. El
Cast se construye en el lifespan y vive en app.state, listo para las rutas.
"""
from __future__ import annotations

import os
from contextlib import asynccontextmanager

import requests
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from muchi.mtg import cast as muchi_cast
from muchi.mtg.http import RateLimited
from muchi.mtg.ports import CommanderNotFound, InventoryUnavailable, QueryFailed

from .routes import router


def build_app(cast=None) -> FastAPI:
    """Arma la app. Acepta un Cast inyectado para los tests, sin tocar la red."""

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        app.state.cast = cast if cast is not None else muchi_cast.build_cast()
        yield

    app = FastAPI(title="Muchi API", lifespan=lifespan)
    app.include_router(router)

    origins = os.getenv("MUCHI_ALLOWED_ORIGINS", "*").split(",")
    app.add_middleware(CORSMiddleware, allow_origins=origins,
                       allow_methods=["*"], allow_headers=["*"])

    def _answer(status: int):
        def handler(request: Request, exc: Exception):
            return JSONResponse(status_code=status, content={"detail": str(exc)})
        return handler

    for exc_type, status in [
        (CommanderNotFound, 404),
        (InventoryUnavailable, 404),
        (QueryFailed, 502),
        (requests.HTTPError, 502),
        (RateLimited, 502),
    ]:
        app.add_exception_handler(exc_type, _answer(status))

    return app


app = build_app()
