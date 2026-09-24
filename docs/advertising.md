[English](advertising-en.md) · [Español](advertising.md)

# La Publicidad, de Punta a Punta

Muchi muestra Anuncios de Google AdSense entre las Ofertas. Este Documento
dice qué hace falta, en qué Orden llega, qué parte se automatiza y qué parte
la decide Google mirando el Sitio.

## Los tres Identificadores, y cuál es cuál

Se confunden seguido, y confundirlos cuesta Días.

| Qué | Ejemplo | Para qué sirve |
| --- | --- | --- |
| **ID de Editor** (Publisher, Cliente) | `ca-pub-6368656861543000` | Identifica la Cuenta. Uno solo, para siempre. |
| **ID de Bloque** (Slot) | `1234567890` | Identifica un Espacio. Uno por cada Lugar que quieras llenar. |
| **ID de Cliente** (Customer) | `3286538783` | Facturación y Soporte. **No entra en Muchi.** |

Ninguno es un Secreto. El de Editor viaja en cada Página y el `ads.txt` lo
declara a Internet entero: esconderlo sería contraproducente. Por eso viajan
como Variables de Entorno y no por Secret Manager, donde vive una sola Cosa:
`MUCHI_API_TOKEN`.

## El Flujo

Cuatro Pasos, y sólo el último es tuyo del todo.

```
1. La Cuenta         creas la Cuenta y registras el Dominio    -> interfaz
2. La Revisión       Google visita el Sitio y lo aprueba       -> esperar
3. El Bloque         creas el Bloque y Google asigna el Slot   -> interfaz
4. El Despliegue     MUCHI_ADSENSE_SLOT=<slot> ./deploy.sh     -> tuyo
```

La API de AdSense sólo lee. `sites` y `adunits` tienen `list` y `get`, nada
más, y `adunits.create` existe pero está reservado a los Publishers de
*AdSense for Platforms*: a una Cuenta normal le responde que no. No hay Guion
que salte los Pasos 1 y 3, y fingir lo contrario sólo esconde la Espera.

Lo que sí se automatiza es la otra Mitad, la que decide el Paso 2.

## Lo que la Revisión mira

Google visita el Dominio registrado —no el `.web.app` de Hosting, el Dominio—
y comprueba cuatro Cosas. Cada una falla en Silencio, y cada Falla cuesta otra
Ronda de Revisión.

- **La Portada responde 200 por https.** Si la Revisión no entra, no aprueba.
- **El `ads.txt` en la Raíz del Dominio** declara al Editor. Lo sirve el BFF en
  [`server/main.py`](../server/main.py); el Dominio tiene que derivar ahí.
- **La Etiqueta `google-adsense-account`** en el `<head>`, en
  [`web/index.html`](../web/index.html).
- **El Cargador `adsbygoogle.js`** apuntando al mismo Editor, en el mismo
  Archivo.

El Identificador aparece en **tres Lugares que no se leen entre sí**: el
`<head>` compilado, el `/ads.txt` del BFF y el `/api/config` que mira el
Front. Si se separan, Muchi pide Publicidad a un Editor y la declara a otro.
Eso no da Error: simplemente no se aprueba, y te enteras Días después.

## Los Guiones

```bash
./setup-ads.sh    # el Onboarding entero: prepara, comprueba y dice qué falta
./check-ads.sh    # sólo el Estado, para repetir mientras esperas
```

`setup-ads.sh` habilita `adsense.googleapis.com`, consigue el Scope del ADC
—preguntando antes, porque reescribe las Credenciales por Defecto de la
Máquina—, le pregunta a AdSense **qué Dominio registraste** y verifica ese, no
uno escrito a mano. Después comprueba las cuatro Cosas de arriba, contrasta
los tres Identificadores, lee las Alertas de la Cuenta y cierra diciendo qué
falta y de quién es. Es idempotente: repetirlo no cambia nada que ya esté bien.

En Windows, `setup-ads.cmd` y `check-ads.cmd` hacen lo mismo.

### Dos Fricciones de la API

Ninguna se diagnostica sola, y las dos ya están resueltas en los Guiones.

**El Scope.** El ADC por Defecto no lleva el de AdSense. Sin él, un `403` que
habla de Scopes y no de cómo arreglarlo:

```bash
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/adsense,https://www.googleapis.com/auth/cloud-platform
```

**El Proyecto de Cuota.** AdSense no es Cloud, pero su API se factura a un
Proyecto de Cloud igual que cualquier otra. Sin la Cabecera, la Consulta se
cobra al Proyecto genérico de gcloud y el `403` culpa a un Proyecto que no es
el tuyo:

```
-H "x-goog-user-project: TU-PROYECTO"
```

## Los Estados, y qué significan

`setup-ads.sh` y `check-ads.sh` muestran el `state` de la Cuenta, el Cliente y
el Sitio. La Espera avanza de a uno:

- **`GETTING_READY`** — Google está revisando. No hay nada que hacer salvo
  esperar, y suele tardar Días.
- **`READY`** — habilita el Paso siguiente. Con el Cliente y el Sitio en
  `READY`, la Interfaz deja crear el Bloque.
- **`Bloques: ninguno`** — todavía no hay Slot que desplegar.

Un Bloqueo de Política no aparece en ningún `state`: vive en las Alertas, que
los Guiones leen aparte. Sin mirarlas, la Espera parece Trámite cuando en
realidad hay algo que arreglar.

## Cómo crear el Bloque

Cuando el Sitio esté en `READY`:

**Anuncios → Por bloque de anuncios → Display → Adaptable.**

El Código que Google muestra trae `data-ad-slot="1234567890"`. Esos diez
Dígitos son el Valor. No copies el `<script>` ni el `<ins>`: los arma
[`GoogleAd.vue`](../web/src/components/GoogleAd.vue).

El mismo Número se puede leer por API, y es el `reportingDimensionId` del
Bloque. `check-ads.sh` lo imprime y cierra con la Línea lista para copiar.

## Cómo decide el Front qué dibujar

Tres Condiciones, en [`App.vue`](../web/src/App.vue) y
[`AdSpot.vue`](../web/src/components/AdSpot.vue):

1. **`environment === 'production'`.** Fuera de Producción el Algoritmo decide
   igual, pero se dibuja un Placeholder: así se prueba la Elección sin
   ensuciar las Métricas de Google.
2. **Cliente y Slot presentes.** Si falta alguno, Muchi cae a la Promoción
   interna en vez de pedir un Anuncio que no puede pedir.
3. **El Reparto.** Con Patrocinador configurado, una de cada cuatro Búsquedas
   lo muestra a él y tres muestran AdSense. La Elección depende del Hash del
   Identificador de la Búsqueda, así que no cambia entre las Consultas
   automáticas de una misma Búsqueda.

Si el Script de Google no carga —un Bloqueador, una Red caída— `GoogleAd.vue`
no deja un Hueco: dice que Muchi sigue buscando Ofertas.

## Las Variables

Todas viajan como Variables de Entorno, vacías por Defecto. Lo que no
exportes, el Front lo omite.

| Variable | Qué hace si falta |
| --- | --- |
| `MUCHI_ADSENSE_CLIENT` | Cae al Editor fijo de `server/main.py`. |
| `MUCHI_ADSENSE_SLOT` | La Publicidad queda apagada; se muestra la Promoción interna. |
| `MUCHI_SPONSOR_NAME` / `_TEXT` / `_URL` | No hay Patrocinador: AdSense se queda con las cuatro de cada cuatro. |

```bash
MUCHI_ADSENSE_SLOT=1234567890 ./deploy.sh
```

Un `.env` entero se exporta con `set -a; . ./.env; set +a` antes de llamar.

`deploy.sh` lo Avisa antes de Subir nada, y nunca Detiene el Despliegue:

```text
~nya?~ Este Deploy va sin Anuncios: MUCHI_ADSENSE_SLOT esta vacio
       ./check-ads.sh dice si Google ya Asigno un Bloque
```

Es un Aviso y no un Error a Propósito: mientras Google Revisa la Cuenta no hay
Slot que Poner, y Bloquear el Despliegue por esa Espera dejaría a Muchi sin
Publicar por algo que no Depende de nadie acá.

## Cuando algo no anda

| Síntoma | Causa probable |
| --- | --- |
| `403` con `ACCESS_TOKEN_SCOPE_INSUFFICIENT` | Falta el Scope en el ADC. |
| `403` nombrando un Proyecto que no es el tuyo | Falta `x-goog-user-project`. |
| `403 The caller does not have permission` | El ID de Editor no es el tuyo. |
| `404` en el `ads.txt` | El BFF sólo lo sirve si hay un Cliente `ca-pub-`. |
| El Sitio lleva Semanas en `GETTING_READY` | Revisa las Alertas y las cuatro Cosas de la Revisión. |
| Hay Slot, pero no aparece el Anuncio | El Servicio no dice `production`, o el Slot no llegó al Despliegue: mira `/api/config`. |
| Hay Slot y `production`, y el Espacio sale vacío | Normal las primeras Horas de un Bloque nuevo, y con un Bloqueador puesto. |
