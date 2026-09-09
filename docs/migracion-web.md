# Del Front Streamlit al Front Vue

Muchi tenía una sola Interfaz posible: Streamlit dibujaba, decidía y guardaba
el Estado en la misma Corrida. Cambiar un Detalle visual obligaba a repintar
la Página completa, y cualquier otra Interfaz —una App, un Widget, un Bot—
habría tenido que reimplementar las Reglas.

Ahora hay dos Piezas y una Frontera clara:

```
Navegador ──► web/  (Vue 3 + Vite)      dibuja, no decide
                │  fetch /api/...
                ▼
           server/  (FastAPI, el BFF)   decide y guarda el Token
                │  Bearer + /v1
                ▼
        API de Muchi (Cloud Run)        busca y persiste las Búsquedas
```

## Por qué un BFF y no llamar la API desde el Navegador

El Código de Seguridad de la API es un Bearer. Un Front que lo llevara en el
Navegador lo estaría publicando: cualquiera abre las Herramientas de
Desarrollo y lo copia. El BFF lo conserva del lado del Servidor y expone solo
lo que el Front necesita.

De paso, las Reglas del Dominio quedan en un solo Lugar. El Tratamiento, el
Muchi Dólar, el Orden por Moneda, la Oferta más barata y el Reparto del
Carrito viven en `server/presenter.py` y reusan `muchi/mtg/`, los mismos
Módulos que usaba `app.py`. El Front recibe JSON ya presentado: Pastillas con
su Texto, Precios con su Moneda y la Oferta marcada como la más barata.

## Qué expone el BFF

| Ruta | Para qué |
| --- | --- |
| `GET /api/config` | Muchi Dólar, Intervalo de Consulta y Límites. Nunca el Token. |
| `GET /api/muchi` | Las Frases del Gato: Saludos, Caricias, Ayuda y Luz. |
| `POST /api/decklist` | Lee la Lista sin gastar una Búsqueda. |
| `POST /api/searches` | Crea la Búsqueda. Recibe el Texto y la Clave de Idempotencia. |
| `GET /api/searches/{id}` | Estado y Ofertas presentadas, en una sola Consulta. |
| `POST /api/searches/{id}/cancel` | Cancela la Búsqueda en curso. |
| `GET /api/searches/{id}/cart?shipping=` | El Carrito en CLP con el Reparto por Tienda. |
| `GET /api/sources` | Estado de las Fuentes. |

Los Errores conservan la Distinción que ya existía en el Dominio: un `502`
trae `retriable: true` y el Front reintenta conservando lo recibido; un `409`
trae `retriable: false` y detiene la Consulta automática, porque esa Búsqueda
ya no existe.

El BFF no guarda Estado. Cada Consulta pregunta a la API, así una Instancia
nueva de Cloud Run atiende igual que la anterior y el Enlace `?search=<id>`
sigue funcionando entre Sesiones y entre Máquinas.

## Qué hace el Front

`web/src/App.vue` orquesta: pide la Configuración, arranca la Consulta cada
`poll_seconds` mientras la Búsqueda está pendiente y la detiene al recibir un
Estado terminal. El Historial de Búsquedas y la Elección de Tema viven en
`localStorage`; antes vivían en la Sesión de Streamlit y se perdían al cerrar.

La Paleta es la misma de `muchi/mtg/style.py`, ahora en `web/src/styles.css`.
El Modo Oscuro no reescribe Reglas: cambia el Valor de las Variables CSS.

## Desarrollo

```bash
cp .env.example .env      # completa MUCHI_API_URL y MUCHI_API_TOKEN
./start-web.sh            # BFF en :8000, Front en http://127.0.0.1:5173
```

Vite reenvía `/api` al BFF, así el Navegador ve un solo Origen y no hay CORS
que configurar, ni en Desarrollo ni en Producción.

## Despliegue en GCP

Un solo Servicio en Cloud Run sirve el Front compilado y el BFF. El
`Dockerfile` compila `web/` con Node y copia el `dist` a la Imagen de Python.

```bash
gcloud builds submit --config cloudbuild.yaml \
  --substitutions=_SERVICE=muchi-web,_REGION=southamerica-east1
```

El Token se monta desde Secret Manager al desplegar; no queda escrito en la
Imagen ni en el Repositorio:

```bash
gcloud run services add-iam-policy-binding muchi-web --region=southamerica-east1 \
  --member=allUsers --role=roles/run.invoker
```

Un Bucket con CDN habría servido el Front más barato, pero entonces el Token
necesitaría un segundo Servicio de todos modos. Un solo Cloud Run con
`min-instances=0` cuesta prácticamente nada mientras nadie lo visite, y evita
tanto el CORS como un segundo Despliegue que mantener sincronizado.

## Qué pasa con Streamlit

`app.py` sigue en pie y funcionando. Las dos Interfaces hablan con la misma
API y comparten el Dominio, así que pueden convivir mientras el Front nuevo se
prueba. Cuando el Vue lo reemplace, se borran `app.py`, `.streamlit/` y
`streamlit` de `requirements.txt`; `muchi/mtg/style.py`, `sprites.py` y las
Partes de `phrases.py` que arman HTML para Streamlit se van con él.
