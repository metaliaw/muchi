"""Small standard-library client for the Streamlit application."""

from __future__ import annotations

import json
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass(frozen=True)
class MuchiAPI:
    base_url: str
    token: str
    timeout_seconds: float = 15.0

    def create_search(
        self,
        cards: list[dict[str, Any]],
        options: dict[str, bool],
        idempotency_key: str,
    ) -> dict[str, Any]:
        payload = {"cards": cards, "options": options}
        return self._request("POST", "/v1/searches", payload, idempotency_key)

    def get_search(self, search_id: str) -> dict[str, Any]:
        return self._request("GET", f"/v1/searches/{search_id}")

    def get_results(self, search_id: str, after: int = 0, limit: int = 50) -> dict[str, Any]:
        query = urllib.parse.urlencode({"after": after, "limit": limit})
        return self._request("GET", f"/v1/searches/{search_id}/results?{query}")

    def cancel_search(self, search_id: str, idempotency_key: str) -> dict[str, Any]:
        return self._request("POST", f"/v1/searches/{search_id}/cancel", {}, idempotency_key)

    def find_offers(self, name: str) -> dict[str, Any]:
        query = urllib.parse.urlencode({"name": name})
        return self._request("GET", f"/v1/cards/offers?{query}")

    def _request(
        self,
        method: str,
        path: str,
        payload: dict[str, Any] | None = None,
        idempotency_key: str | None = None,
    ) -> dict[str, Any]:
        data = json.dumps(payload).encode() if payload is not None else None
        headers = {"Authorization": f"Bearer {self.token}", "Content-Type": "application/json"}
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        request = urllib.request.Request(self.base_url.rstrip("/") + path, data=data, headers=headers, method=method)
        with urllib.request.urlopen(request, timeout=self.timeout_seconds) as response:
            return json.load(response)
