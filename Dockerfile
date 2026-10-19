# Un solo Servicio en Cloud Run: el Front compilado y el BFF que lo sirve.
# Compilar el Front acá evita subir node_modules y deja el Despliegue
# reproducible: la misma Imagen se construye igual en cualquier Máquina.
# La Imagen Repite el Árbol del Repositorio —bff/ junto a frontend/dist— para
# que el BFF Encuentre el Front en la misma Ruta en los dos Lados.
FROM node:22-alpine AS web
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci || npm install
COPY frontend/ ./
RUN npm run build

FROM python:3.11-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app/bff

COPY bff/requirements-web.txt ./
# El Front no arrastra el Scraping antiguo: solo el Dominio, el Cliente HTTP
# de la API y el BFF.
RUN pip install --no-cache-dir -r requirements-web.txt \
    "requests>=2.32" "pyyaml>=6.0" "python-dotenv>=1.0"

COPY bff/muchi/ ./muchi/
COPY bff/server/ ./server/
COPY bff/config/ ./config/
COPY bff/constants/ ./constants/
COPY --from=web /frontend/dist /app/frontend/dist

ENV MUCHI_ENV=production PORT=8080
EXPOSE 8080
# Cloud Run entrega el Puerto en $PORT y espera que el Proceso escuche ahí.
CMD ["sh", "-c", "uvicorn server.main:app --host 0.0.0.0 --port ${PORT:-8080}"]
