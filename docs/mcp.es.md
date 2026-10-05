[English](mcp.md) · **Español**

# Guía de Uso del MCP Público de MUCHI

MUCHI expone su Catálogo y las Ofertas de las Tiendas por el Model Context
Protocol (MCP). Un Agente compatible puede buscar Cartas desde un Chatbot; no
necesita abrir el Sitio ni conocer el Token de MUCHI API.

## Conectar un Chatbot

En la configuración MCP del Chatbot, agrega un Servidor Remoto con estos Datos.
Los nombres exactos de los Campos cambian según el Cliente:

| Campo | Valor |
| --- | --- |
| Nombre | MUCHI |
| Transporte | Streamable HTTP |
| URL | `https://muchitcg.cl/mcp/` |
| Autenticación | Ninguna |

El endpoint es público y no pide cuenta, API key ni Token. La conexión se
configura una vez; después puedes pedirle búsquedas en la conversación. El
Chatbot debe admitir Servidores MCP Remotos por Streamable HTTP.
El BFF mantiene `MUCHI_API_TOKEN` en el Servidor; nunca se lo entregues al
Chatbot.

Al conectar, confirma que aparezcan las herramientas `search_cards` y
`get_search_results`.

Para Desarrollo local, conecta a `http://127.0.0.1:8000/mcp/` después de
iniciar el BFF según el [README](../README.es.md).

## Probar una Búsqueda Pública

Usa una carta para comprobar que el Servidor inicia la Búsqueda y devuelve sus
Ofertas. En cualquier Cliente MCP, llama a `search_cards` con estos Argumentos:

```json
{"decklist":"1 Sol Ring","game":"magic","match":"exact","kind":"single"}
```

Guarda el `search_id` de la Respuesta. La Búsqueda puede empezar con Estado
`queued`; consulta `get_search_results` con ese ID, `after: 0`, `match: exact` y
`sort_by: price_asc`. Si `state.done` sigue en `false`, espera unos Segundos y
vuelve a consultar. Al terminar, comprueba que `items` incluya Sol Ring y que
`state.done` sea `true`. El Resumen informa cuántas Ofertas y Tiendas encontró.
El Precio puede cambiar a medida que las Tiendas actualizan sus Catálogos.

## Pedir una Búsqueda

Especifica MUCHI y las Cartas. Si no das más Opciones, se usa Magic, coincidencia
exacta y Cartas sueltas.

```text
Busca en MUCHI estas cartas de Magic:
1 Sol Ring
4 Lightning Bolt
2 Counterspell
Espera a que termine la búsqueda, recoge todas las páginas y ordénalas
por precio ascendente. Devuelve tienda, precio, stock y enlace.
```

Para productos sellados, dilo expresamente:

```text
Busca en MUCHI una caja sellada de [nombre del producto].
Espera los resultados finales y ordénalos por precio ascendente.
```

No necesitas escribir la URL en cada petición. Si el Chatbot tiene otros
Servidores conectados, decir «en MUCHI» ayuda a elegir este.

## Cómo Continúa la Búsqueda

`search_cards` inicia una Búsqueda en la API de MUCHI y devuelve un
`search_id`. El Trabajo continúa en el Servidor aunque el Chatbot todavía no
consulte el resultado.

El Agente llama a `get_search_results` con ese Identificador. Mientras
`state.done` sea `false`, espera un momento y vuelve a consultar. Cuando
`has_more` sea `true`, pasa el `cursor` recibido como `after` para recoger la
Página siguiente. Termina cuando la Búsqueda acabó y ya no quedan Páginas.

Puedes pedir al Chatbot que espere hasta terminar. Si la conversación se
interrumpe, guarda el `search_id` y pídele que retome la Búsqueda mientras la
API todavía conserve sus Resultados.

## Orden de las Ofertas

Pide el orden que necesitas. `get_search_results` acepta:

- `price_asc`: de menor a mayor Precio CLP; es el orden inicial.
- `price_desc`: de mayor a menor Precio CLP.

El orden se aplica dentro de cada Tipo de Carta, y las Ofertas sin Conversión
a CLP quedan al final. La Respuesta incluye Tienda, Precio, Moneda, Stock,
Enlace y señales de Precio sospechoso cuando la Fuente las informa.

MCP no recibe la ubicación de quien pregunta. El Sitio puede ordenar por
cercanía con el permiso del Navegador; esa ubicación no se envía a MUCHI. MCP
puede ordenar por Precio, no por «cerca de mí».

## Herramientas

### `search_cards`

Inicia una Búsqueda y devuelve su `search_id` y Estado inicial. La Búsqueda se
procesa en segundo plano y queda persistida por la API de MUCHI.

| Argumento | Tipo | Valor inicial | Uso |
| --- | --- | --- | --- |
| `decklist` | texto | — | Una Carta por Línea; acepta `4 Sol Ring` o `Sol Ring`. |
| `game` | texto | `magic` | Juego reconocido por MUCHI API. |
| `match` | `exact` o `includes` | `exact` | Cómo comparar el Nombre de la Carta. |
| `kind` | `single` o `sealed` | `single` | Cartas sueltas o Productos Sellados. |

La Lista admite hasta 100 entradas, con Cantidades de 1 a 99. Las líneas que
no se puedan interpretar hacen fallar la herramienta; no se descartan en
silencio.

Ejemplo de `decklist`:

```text
1 Sol Ring
4 Lightning Bolt
2 Counterspell
```

La Respuesta incluye el Identificador, el Estado, el Progreso, las Entradas
encontradas y las que fallaron. Guarda `search_id` para leer las Ofertas.

### `get_search_results`

Lee el Estado actual, hasta 50 Entradas de la Lista y una Lista plana de sus
Ofertas. `item_position` relaciona cada Oferta con su Entrada.

| Argumento | Tipo | Valor inicial | Uso |
| --- | --- | --- | --- |
| `search_id` | texto | — | Identificador devuelto por `search_cards`. |
| `after` | entero | `0` | Cursor de la Página anterior. |
| `match` | `exact` o `includes` | `exact` | Usa el Modo que enviaste al crear la Búsqueda. |
| `sort_by` | `price_asc` o `price_desc` | `price_asc` | Ordena por Precio CLP dentro de cada Tipo de Carta. |

La Página contiene hasta 50 Entradas de la Lista y sus Ofertas. `item_position`
relaciona cada Oferta con su Entrada. El cursor sirve para pedir la Página
siguiente.

## Límites y Seguridad

- MCP solo inicia Búsquedas y lee sus Resultados; no crea Pedidos ni Compra.
- Las Búsquedas usan las validaciones, la presentación y la API ya existentes.
- Cualquier Agente que alcance la URL puede iniciar Búsquedas. La API interna y
  su Token siguen del lado Servidor.
- El Transporte acepta los Hosts del Sitio y las Direcciones locales de
  Desarrollo para frenar solicitudes con un Host ajeno.
- Firebase Hosting reenvía el Host de Servicio de Cloud Run. Si cambia la URL
  del Servicio, actualiza también ese Host permitido en `MCP_HOSTS` de
  `server/mcp.py`.

La API conserva los Resultados y aplica su propio Vencimiento. Un Agente debe
guardar el Identificador recibido mientras los Resultados sigan disponibles.
