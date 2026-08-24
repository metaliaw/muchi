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
# Por defecto apunta al repo, no a un mail: esto corre en Streamlit Cloud y vive
# en un repo publico, y una direccion de correo ahi la cosechan los bots de spam
# en minutos. Quien tenga una queja abre un issue.
#
# Si preferis que te escriban por mail, no lo escribas aca: exportalo como
#     MUCHI_CONTACTO="tu-mail@ejemplo.cl"
# (en Streamlit Cloud: Settings -> Secrets/Variables).
CONTACTO = os.getenv("MUCHI_CONTACTO", "https://github.com/tu-usuario/muchi")
USER_AGENT = f"Muchi/0.1 (uso personal; +{CONTACTO})"


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

    def _esperar_turno(self, host: str) -> None:
        with self._lock:
            ahora = time.monotonic()
            falta = self._last.get(host, 0.0) + self.min_interval - ahora
            if falta > 0:
                time.sleep(falta)
                ahora = time.monotonic()
            self._last[host] = ahora

    def get(self, url: str, *, params=None, headers=None, stream=False, timeout=None):
        host = urlparse(url).netloc
        backoff = 5.0
        ultimo_error: Exception | None = None

        for _ in range(self.max_retries + 1):
            self._esperar_turno(host)
            try:
                r = self._s.get(
                    url, params=params, headers=headers or {},
                    stream=stream, timeout=timeout or self.timeout,
                )
            except requests.RequestException as e:
                ultimo_error = e
                time.sleep(backoff)
                backoff *= 2
                continue

            if r.status_code == 429:
                espera = r.headers.get("Retry-After")
                r.close()
                time.sleep(min(float(espera) if espera and espera.isdigit() else backoff, 60))
                backoff *= 2
                ultimo_error = RateLimited(f"429 en {url}")
                continue

            if r.status_code >= 500:
                r.close()
                ultimo_error = requests.HTTPError(f"{r.status_code} en {url}")
                time.sleep(backoff)
                backoff *= 2
                continue

            r.raise_for_status()
            return r

        raise ultimo_error or RuntimeError(f"no se pudo obtener {url}")
