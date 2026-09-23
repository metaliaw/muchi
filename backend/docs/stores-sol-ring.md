# Tiendas Candidatas desde Sol Ring

[English](candidate-stores-sol-ring.md) | **Español**

Verificación: 7 de septiembre de 2026. Búsqueda real en GCP: `search_1bad0a3b9c8aacc035c23576`.
La búsqueda terminó `completed`, con una carta encontrada y cero errores. Devolvió **208 ofertas**: 198 de scry.cl, 4 directas de La Cripta y 6 de Moxfield.

API: https://muchi-serve-api-c2ce6c7oga-rj.a.run.app. Salud: HTTP 200. Worker sin autenticación: HTTP 403. El flujo con Cloud Tasks terminó correctamente en 2,71 segundos usando la caché de la consulta directa anterior.

**OnPlay, las tres Tiendas Shopify y las cuatro Tiendas Jumpseller están habilitadas en `config/stores.yaml`.** Las cinco restantes requieren trabajo adicional. La Cripta y El Wombat Rabioso ya están configurados.

Los conteos siguientes corresponden a ofertas presentes en esta búsqueda de Scry, no al tamaño total de cada catálogo ni a un stock confirmado.

| Tienda | Dominio | Ofertas | Plataforma | Próximo Paso |
| --- | --- | ---: | --- | --- |
| OnPlay Singles | [onplay.cl](https://onplay.cl) | 8 | WooCommerce | Agregada a stores.yaml |
| Rhystic Bazaar | [rhysticbazaar.cl](https://rhysticbazaar.cl) | 17 | WooCommerce | Ajustar Coincidencia de Nombres |
| CardNexus | [cardnexus.cl](https://cardnexus.cl) | 26 | WooCommerce | Resolver Acceso del Lector |
| Game of Magic Singles | [gameofmagicsingles.cl](https://gameofmagicsingles.cl) | 14 | Shopify | Agregada a stores.yaml |
| Collector Center | [singles.collectorcenter.cl](https://singles.collectorcenter.cl) | 16 | Shopify | Agregada a stores.yaml |
| CardSouls | [www.cardsouls.cl](https://www.cardsouls.cl) | 16 | Shopify | Agregada a stores.yaml |
| Cartas La Fortaleza | [www.cartaslafortaleza.cl](https://www.cartaslafortaleza.cl) | 3 | Jumpseller | Agregada a stores.yaml |
| ChronoMagic | [www.chronomagic.cl](https://www.chronomagic.cl) | 1 | Jumpseller | Agregada a stores.yaml |
| GameQuest | [gamequest.cl](https://gamequest.cl) | 3 | Jumpseller | Agregada a stores.yaml |
| Magic4Ever | [www.magic4ever.cl](https://www.magic4ever.cl) | 3 | Jumpseller | Agregada a stores.yaml |
| CatLotus | [catlotus.cl](https://catlotus.cl) | 39 | Sin API Compatible Verificada | Investigar Catálogo |
| Dominio Arcano | [dominioarcano.cl](https://dominioarcano.cl) | 2 | Sin API Compatible Verificada | Investigar Catálogo |
| HunterCard TCG | [www.huntercardtcg.com](https://www.huntercardtcg.com) | 1 | Sin API Compatible Verificada | Revisar URL y Catálogo |

## Evidencia por Tienda

- **OnPlay Singles**: El lector real completó la consulta: 8 ofertas en CLP, las 8 con disponibilidad reportada por el catálogo. [Producto observado](https://onplay.cl/product/sol-ring-5/).
  [Catálogo consultado](https://onplay.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **Rhystic Bazaar**: API pública accesible. Usa nombres como «Sol Ring — Near Mint»; la igualdad exacta actual descarta todos estos resultados. [Producto observado](https://rhysticbazaar.cl/product/sol-ring-near-mint-7/).
  [Catálogo consultado](https://rhysticbazaar.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **CardNexus**: Una consulta exploratoria devolvió 100 productos de nombre exacto en CLP. El lector Go real recibió HTTP 403 con comprobación de navegador; paginación completa sin verificar. [Producto observado](https://cardnexus.cl/product/sol-ring-36/).
  [Catálogo consultado](https://cardnexus.cl/wp-json/wc/store/v1/products?search=Sol%20Ring&per_page=100).
- **Game of Magic Singles**: Lector Shopify habilitado, con búsqueda paginada y stock por variante. [Producto observado](https://gameofmagicsingles.cl/products/sol-ring-2683-secret-lair-drop-series?variant=54369284817206).
  [Producto JSON verificado](https://gameofmagicsingles.cl/products/sol-ring-2683-secret-lair-drop-series.js).
- **Collector Center**: Lector Shopify habilitado, con búsqueda paginada y stock por variante. [Producto observado](https://singles.collectorcenter.cl/products/stcg-card-92a2e8c4-dc36-4443-b95e-3f8f26d4b84b?variant=51967974768928).
  [Producto JSON verificado](https://singles.collectorcenter.cl/products/stcg-card-92a2e8c4-dc36-4443-b95e-3f8f26d4b84b.js).
- **CardSouls**: Lector Shopify habilitado, con búsqueda paginada y stock por variante. [Producto observado](https://www.cardsouls.cl/products/sol-ring-0356-fic-356?variant=51342901575903).
  [Producto JSON verificado](https://www.cardsouls.cl/products/sol-ring-0356-fic-356.js).
- **Cartas La Fortaleza**: Plataforma identificada en la página de producto. La ruta WooCommerce devuelve 404. [Producto observado](https://www.cartaslafortaleza.cl/sol-ring-ingles-nm-cma).
- **ChronoMagic**: Plataforma identificada en la página de producto. La ruta WooCommerce devuelve 404. [Producto observado](https://www.chronomagic.cl/sol-ring-lcc-313-en-nm).
- **GameQuest**: Plataforma identificada en la página de producto. La ruta WooCommerce devuelve 404. [Producto observado](https://gamequest.cl/sol-ring-espanol-nm-afc).
- **Magic4Ever**: Plataforma identificada en la página de producto. La ruta WooCommerce devuelve 404. [Producto observado](https://www.magic4ever.cl/sol-ring-7?variant=121035117).
- **CatLotus**: Página de producto accesible; la ruta WooCommerce devuelve 404. [Producto observado](https://catlotus.cl/cartas/C21/Sol%20Ring?name=Sol+Ring&set=C21).
- **Dominio Arcano**: Página de producto accesible. La ruta probada devuelve 200, pero no el JSON de productos WooCommerce esperado. [Producto observado](https://dominioarcano.cl/mtg/single/sol-ring-c20-252).
- **HunterCard TCG**: La URL de producto entregada por Scry devuelve 404. La ruta probada no devuelve un catálogo WooCommerce compatible. [Producto observado](https://www.huntercardtcg.com/producto/sol-ring-showcase-promo-316-ingles/).

## Entrada Agregada para OnPlay

Entrada habilitada bajo `stores:` en `config/stores.yaml`.

```yaml
  onplay.cl:
    name: "OnPlay Singles"
    platform: woocommerce
    unavailable_selectors:
      - "div.product.outofstock"
      - "p.stock.out-of-stock"
    unavailable_text:
      - "Sold out"
      - "Agotado"
    scope_selector: "div.product"
    timeout_seconds: 10
    allow_redirects: true
    enabled: true
```

La prueba confirmó catálogo, precios CLP y disponibilidad reportada por la API WooCommerce. Los selectores siguen el patrón existente de La Cripta; no se verificó una página agotada de OnPlay. Las comprobaciones HTML de stock conservan las limitaciones del lector actual.

## Vendedores de Marketplace

La búsqueda también devolvió **46 ofertas de 20 vendedores** en `marketplace.scry.cl`. Sus enlaces no aportan un dominio propio que permita agregarlos como catálogos WooCommerce independientes.

| Vendedor | Ofertas |
| --- | ---: |
| Bazar de Waldosky | 1 |
| Cartas del Nexo | 1 |
| El Bazar de Olguis | 1 |
| Fallen Angel | 2 |
| Grafitos | 8 |
| Ineko Card Shop | 4 |
| Kydroukair | 3 |
| La resistencia de Gerrard | 1 |
| Magic Master | 1 |
| Mana Surge | 5 |
| Mox & Mana | 1 |
| Naipes Embrujados TCG | 1 |
| Posada Canto de Guerra | 3 |
| The Stack | 4 |
| Tiendálogo | 1 |
| Wither and Bloom | 1 |
| Woodedcard | 1 |
| Zanahoria | 2 |
| laipp's store | 2 |
| moxes | 3 |

Los resultados son una muestra de una carta y del estado observado en esta fecha; no constituyen un listado exhaustivo de tiendas de Chile.

## Lector Shopify

Consulta `/search` con el nombre entre comillas, tipo producto y filtro de disponibilidad. Recorre la paginación y lee `/products/{handle}.js` para obtener variantes disponibles, precios y opciones. Consulta `/cart.js` para verificar la moneda; convierte los importes de centésimas a unidades monetarias. Acepta sufijos de edición y descarta nombres de otras cartas. Cada oferta conserva una URL con `?variant=ID`; la comprobación de stock consulta esa variante, incluso si está agotada.

Prueba real optativa: `MUCHI_TEST_SHOPIFY_LIVE=1 go test ./internal/stores/shopify -run TestLiveSolRing -v`. Estos cambios de configuración y lector requieren un nuevo despliegue para llegar a GCP.

Validación directa con el lector Go: **74 ofertas disponibles de Sol Ring**, repartidas en Game of Magic Singles (34), Collector Center (19) y CardSouls (21). Las tres consultas terminaron correctamente; todas las ofertas indicaron CLP y se comprobó una variante disponible por tienda. Duración total: 28,36 segundos, sin la coordinación Firestore de producción. Estos conteos corresponden a esta prueba directa, no a la búsqueda GCP histórica de arriba.

## Lector Jumpseller

Cartas La Fortaleza, ChronoMagic, GameQuest y Magic4Ever usan `platform: jumpseller`. El lector recorre `/api/search/{nombre}?page=...` con los encabezados AJAX del escaparate, filtra nombres de cartas (incluidos sufijos de edición, idioma y condición) y consulta únicamente los productos coincidentes. Jumpseller hace búsquedas aproximadas: las comillas no eliminan resultados ajenos y algunas consultas tienen muchas páginas. No se corta la búsqueda al encontrar una página sin coincidencias. La paginación identifica productos por ID, porque los enlaces pueden repetirse entre productos distintos. Termina cuando la API devuelve una página vacía y rechaza páginas sin ningún ID nuevo.

Los temas de estas cuatro tiendas publican `product-form-json` y `product-json` en HTML. El lector verifica la moneda en los metadatos del producto, lee los precios en unidades monetarias y resta el descuento publicado. Las variantes conservan `?variant_id=ID`, idioma, condición y acabado. La disponibilidad combina estado y cantidad o stock ilimitado: un estado `available` con cantidad cero sigue siendo agotado.

Las consultas se serializan dentro de cada proceso y se espacian cuatro segundos y mantienen los reintentos y la coordinación de fuentes existentes. Los errores, incluidos HTTP 429, se propagan para evitar guardar una búsqueda incompleta como resultado definitivo. Las búsquedas grandes pueden tardar varios minutos.

Contrato de referencia: [Liquid de Jumpseller](https://jumpseller.com/support/liquid/) y [Búsqueda de Productos](https://jumpseller.com/support/search/). La API pública requiere los encabezados del escaparate (`X-Requested-With` y `Referer`). Las consultas sin ellos respondieron HTTP 403. La búsqueda usa esta API y la lectura de precios y stock usa los datos JSON publicados en las páginas de producto.

Prueba optativa: `MUCHI_TEST_JUMPSELLER_LIVE=1 go test ./internal/stores/jumpseller -run TestLiveSolRing -v -timeout 90m`. Los fixtures de las cuatro tiendas conservan fragmentos públicos relevantes para comprobar precios y stock sin red. Cambios pendientes de despliegue en GCP.

El plazo de despliegue y de las tareas HTTP se amplió a 1800 segundos para permitir el recorrido completo con pausas. El flujo local asíncrono evita el límite de escritura HTTP de la API local. [Cloud Tasks admite plazos HTTP de hasta 30 minutos](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks).

### Estado de la Validación Jumpseller

Las consultas HTML completas confirmaron 17 ofertas disponibles en Cartas La Fortaleza, 2 en ChronoMagic y 3 en GameQuest, todas en CLP. La versión definitiva con búsqueda AJAX confirmó nuevamente las 2 de ChronoMagic. Magic4Ever superó la página 60 usando IDs de producto, pero respondió HTTP 429 al probar simultáneamente otras tiendas; no quedó validada su búsqueda completa. Las últimas consultas AJAX de Cartas La Fortaleza y GameQuest también recibieron 429.

La pausa final de cuatro segundos y la serialización local se agregaron después de esa prueba. No se repitió otro barrido completo contra el límite activo. La coordinación Firestore existente sigue operando por dominio; la serialización local no garantiza un límite global entre instancias. Los errores 429 siguen visibles para el agregador y evitan almacenar resultados agregados incompletos en caché.
