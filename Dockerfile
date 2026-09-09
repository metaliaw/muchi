# Un solo Servicio en Cloud Run: el Front compilado y el BFF que lo sirve.
# Compilar el Front acá evita subir node_modules y deja el Despliegue
# reproducible: la misma Imagen se construye igual en cualquier Máquina.
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci || npm install
COPY web/ ./
RUN npm run build

FROM python:3.11-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app

COPY requirements-web.txt ./
# El Front no arrastra el Scraping antiguo: solo el Dominio, el Cliente HTTP
# de la API y el BFF.
RUN pip install --no-cache-dir -r requirements-web.txt \
    "requests>=2.32" "pyyaml>=6.0" "python-dotenv>=1.0"

COPY muchi/ ./muchi/
COPY server/ ./server/
COPY config/ ./config/
COPY constants/ ./constants/
COPY --from=web /web/dist ./web/dist

ENV MUCHI_ENV=production PORT=8080
EXPOSE 8080
# Cloud Run entrega el Puerto en $PORT y espera que el Proceso escuche ahí.
CMD ["sh", "-c", "uvicorn server.main:app --host 0.0.0.0 --port ${PORT:-8080}"]
