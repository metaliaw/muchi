# Muchi en un Contenedor: Dependencias primero, Codigo despues.
#
# El orden importa. requirements.txt cambia poco y el codigo cambia siempre,
# asi que la capa de pip se cachea entre deploys y solo se recompila cuando
# una dependencia se mueve de verdad.
#
# Esta Rama sirve el Front de Streamlit. El Front Vue vive en
# experimental-vue, con su propio Dockerfile de dos Etapas y su BFF.
FROM python:3.12-slim

# PYTHONUNBUFFERED manda los logs a Cloud Logging apenas se escriben; sin el,
# Python los guarda en un buffer y el ultimo grito antes de morir se pierde.
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

COPY requirements.txt ./
RUN pip install --requirement requirements.txt

COPY . .

# Cloud Run elige el Puerto y lo anuncia en PORT; 8080 es su Default y sirve
# para correr esta misma Imagen a mano. La direccion va en 0.0.0.0 porque el
# Contenedor recibe Trafico de afuera, no de si mismo.
ENV PORT=8080
EXPOSE 8080

CMD exec streamlit run app.py \
    --server.port "$PORT" \
    --server.address 0.0.0.0 \
    --server.headless true \
    --server.runOnSave false \
    --browser.gatherUsageStats false
