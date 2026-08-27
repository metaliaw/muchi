"""Cliente HTTP educado: un request por host cada N segundos, respeta 429/Retry-After.

Esto no es opcional. scry.cl ya nos devolvio un 429 durante el reconocimiento,
y las tiendas son negocios chicos: si las martillamos, nos bloquean (con razon).
"""
from __future__ import annotations

import os
import threading
import time
from urllib.parse import urlparse

import requests

# Via de contacto que viaja en el User-Agent de cada request. Existe para que
# scry.cl o una tienda puedan avisarte en vez de bloquearte de una.
#
# Antes apuntaba al repo, para no dejar un correo a la vista de los bots de
# spam. Ya no sirve: el repo es privado y ahi no puede abrir un issue nadie de
# afuera, asi que la URL seria una promesa muerta --- peor que no poner nada.
#
# El canal se define fuera del codigo, que ademas es donde corresponde:
#     MUCHI_CONTACTO="donde-te-lleguen@ejemplo.cl"
# (en Streamlit Cloud: Settings -> Secrets/Variables).
#
# Sin la variable, el User-Agent igual identifica a Muchi, pero no ofrece por
# donde reclamar. Se puede salir asi a mirar precios; no se deberia dejar asi
# corriendo seguido contra tiendas que son negocios chicos.
CONTACT = os.getenv("MUCHI_CONTACTO", "")
USER_AGENT = (f"Muchi/0.1 (uso personal; +{CONTACT})" if CONTACT
              else "Muchi/0.1 (uso personal)")


class RateLimited(RuntimeError):
    """El servidor nos pidio bajar el ritmo mas veces de las que reintentamos."""


class PoliteSession:
    def __init__(self, min_interval: float = 1.5, timeout: float = 30.0, max_retries: int = 3):
        self.min_interval = min_interval
        self.timeout = timeout
        self.max_retries = max_retries
        self._last: dict[str, float] = {}
        self._lock = threading.Lock()
        self._s = requests.Session()
        self._s.headers.update({
            "User-Agent": USER_AGENT,
            "Accept-Language": "es-CL,es;q=0.9",
        })

    def _wait_turn(self, host: str) -> None:
        with self._lock:
            now = time.monotonic()
            remaining = self._last.get(host, 0.0) + self.min_interval - now
            if remaining > 0:
                time.sleep(remaining)
                now = time.monotonic()
            self._last[host] = now

    def get(self, url: str, *, params=None, headers=None, stream=False, timeout=None):
        host = urlparse(url).netloc
        backoff = 5.0
        last_error: Exception | None = None

        for _ in range(self.max_retries + 1):
            self._wait_turn(host)
            try:
                r = self._s.get(
                    url, params=params, headers=headers or {},
                    stream=stream, timeout=timeout or self.timeout,
                )
            except requests.RequestException as e:
                last_error = e
                time.sleep(backoff)
                backoff *= 2
                continue

            if r.status_code == 429:
                wait = r.headers.get("Retry-After")
                r.close()
                time.sleep(min(float(wait) if wait and wait.isdigit() else backoff, 60))
                backoff *= 2
                last_error = RateLimited(f"429 en {url}")
                continue

            if r.status_code >= 500:
                r.close()
                last_error = requests.HTTPError(f"{r.status_code} en {url}")
                time.sleep(backoff)
                backoff *= 2
                continue

            r.raise_for_status()
            return r

        raise last_error or RuntimeError(f"no se pudo obtener {url}")
