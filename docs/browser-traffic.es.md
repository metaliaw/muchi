[English](browser-traffic.md) · [Español](browser-traffic.es.md)

# El Tráfico que no Pagamos

Septiembre 2026 · el Patrón, no el Truco

Muchi Consulta Tiendas ajenas para Existir. Cada Consulta Cuesta: Tiempo de
Cola, Ancho de banda, una Instancia encendida, y sobre todo una Visita más que
una Tienda Recibe de nosotros. El Costo no es la Factura de Google —Cloud Run
Escala a cero—, es la Cadena entera.

Pero no todas las Consultas las Tenemos que Hacer nosotros. Algunas las Puede
Hacer el Navegador de quien Compra, directo, sin Pasar por acá. Este Documento
Cuenta cuándo Vale la pena Moverlas ahí, qué lo Hace posible, y dónde está la
Raya que no Cruzamos.

## Las tres Billeteras

Cada Petición que Existe en Muchi la Paga alguien:

| Quién | Qué Cuesta | Ejemplo |
| --- | --- | --- |
| **Nosotros** | Instancia, Cola, nuestra IP frente a la Tienda | Buscar Precios en veinte Fuentes |
| **El Navegador** | Nada nuestro; una Pestaña abierta y la Conexión de quien ya Estaba ahí | Pedir la Foto de una Carta |
| **Nadie** | El Dato ya Estaba | Releer una Búsqueda Guardada |

La tercera es la mejor y por eso la última Búsqueda se Guarda en
`localStorage`: Recargar la Página no Manda a nadie a las Tiendas otra vez.
Cuando el Dato no Existe todavía, la Pregunta es entre la primera y la segunda.

## Por qué la Arquitectura lo Permite

El Front y el BFF Viajan en una sola Imagen y un solo Origen. Eso se Hizo por
el Token —un Front que lo Llevara lo estaría Publicando— y Trajo de regalo que
el Navegador no Tenga que Preguntarnos a nosotros por lo que Puede Pedir solo.

Lo que el Navegador Alcanza lo Decide el otro Lado, no nosotros: una Tienda que
Manda `Access-Control-Allow-Origin` Está Diciendo que su Dato es público y que
cualquiera lo Puede Leer desde cualquier Página. No es un Agujero que
Encontramos; es una Puerta que la Tienda Abrió.

Donde no la Abrió, no Entramos. Un `fetch` sin CORS Vuelve opaco: sin Cuerpo,
sin Estado, sin Cabeceras. Eso no se Puede Rodear desde una Página web, y el
Documento de la Confirmación de Stock Lista las cuatro Plataformas donde se
Intentó y se Chocó.

## Lo que ya Viaja por el Navegador

No es una Idea nueva en este Repositorio; ya Estaba Pasando sin Nombre:

- **Las Fotos de las Ofertas.** `offer.image` es una URL del CDN de la Tienda y
  Entra derecho a un `<img>`. Nunca Pasa por nosotros. Una Búsqueda de cien
  Cartas Carga cien Imágenes que no nos Cuestan un byte.
- **El Arte de las Cartas.** El BFF Pregunta a Scryfall solo la Ficha; la
  Imagen la Trae el Navegador del CDN de Scryfall.
- **AdSense.** El Script y el Anuncio los Sirve Google al Navegador. Nosotros
  Mandamos un Identificador, nada más.
- **La Confirmación de Stock.** La más reciente y la única que Elegimos a
  Propósito: el Navegador Lee `/products/<handle>.js` de las Tiendas Shopify y
  nos Manda solo la Respuesta. Está Contada entera en
  [Por qué la Re-verificación de Stock es Opcional](optional-reverification.es.md).

## La Regla

Una Petición se Mueve al Navegador cuando las cuatro se Cumplen:

1. **La Tienda la Abrió.** Hay CORS, o es un `<img>`, o un `<script>` que el
   Dueño Publica para eso. No hay Proxy, no hay Rodeo.
2. **Alguien la Pidió.** Un Toque, no un Ciclo. Una Petición que Sale sola
   Convierte la Pestaña de quien Compra en un Robot que no Encendió.
3. **Tiene Techo.** `browser_check_limit` por Carta, y Corta en el primer Sí.
   Sin Techo, una Lista de cien Cartas Manda quinientas Peticiones desde la
   Casa de alguien.
4. **Sirve a quien la Hace.** La Consulta Contesta la Pregunta de esa misma
   Persona, en ese mismo Momento. No Recolectamos para nosotros.

La cuarta es la que Separa esto de Usar Visitantes como Flota de Scraping.
Quien Aprieta "Confirmar Stock" Quiere saber si la Carta Está; que de paso no
nos Cueste una Visita es la Consecuencia, no el Motivo. Si algún día el Motivo
se Da vuelta —Pedirle al Navegador algo que solo nos Sirve a nosotros— esto
Dejó de ser este Patrón y Pasó a ser otra cosa.

## Lo Medido

Una Búsqueda real de Sol Ring, 201 Ofertas de 18 Hosts:

- El Navegador Alcanza **una de cada cinco** Ofertas (las Shopify).
- Con Corte en el primer Sí: **una sola Consulta**, medio Segundo, para Mover
  la Corona de una Oferta agotada a una Confirmada.
- Sin Corte serían 82. El Techo Importa más que el Alcance.

## Lo que no Hacemos

- **No Rodeamos un Desafío de Bot.** `cartasmagicsur.cl` Contesta `429` a todo
  Cliente que no Sea un Navegador con JavaScript. Un Navegador de verdad lo
  Pasaría. No lo Usamos para eso: esa Tienda Puso una Puerta y la Puerta Vale
  también para nosotros. Su Oferta se Queda en `unknown` para siempre.
- **No Consultamos Fichas de la Comunidad.** Los que Venden en el Marketplace
  de scry son Usuarios de su Plataforma, no Tiendas. Sus Ofertas ya se Excluyen
  desde la Fuente, y Comprobarles el Stock sería Visitar la Página de una
  Persona que no Puso una Vitrina.
- **No Guardamos lo que Envejece.** Un Stock confirmado Vuelve con su Hora y
  Caduca solo. Guardar el Sí a secas sería Convertir un Ahorro en una Mentira.
- **No Inventamos un No.** CORS cerrado, Red caída o JSON que no lo era Vuelven
  nulos. Un Agotado que nadie Dijo es peor que no Saber.

## Dónde más Podría Aplicarse

- **Precio en vivo de una Oferta.** El mismo JSON de Shopify Trae el Precio.
  Confirmar que no Cambió Cuesta lo mismo que Confirmar el Stock, y Entrar al
  Carrito con un Precio viejo es el otro Modo de que una Compra Falle.
- **WooCommerce, si Abren la Puerta.** Su Store API ya Sirve `is_in_stock` en
  JSON; lo único que Falta es una Cabecera. Son ~2.400 Ofertas del Catálogo.
  Eso no se Programa: se Pide.

## Lo que Queda Abierto

- Medir qué Fracción de las Confirmaciones Cambia algo. Si casi ninguna Mueve
  la Corona, el Botón Cuesta Atención sin Devolver Certeza.
- Decidir si el Patrón Merece un Aviso visible. Hoy quien Aprieta el Botón no
  Sabe que su Navegador va a Hablarle a cinco Tiendas. Es poco Tráfico y es
  para él, pero Decirlo Cuesta una Línea.
