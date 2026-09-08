# Cómo compartir el Stock de tu Tienda con Muchi

Si tienes una Tienda de Cartas y quieres que sus Ofertas aparezcan en Muchi,
puedes compartir tu Inventario mediante Listas de Moxfield o desde tu sitio web.
Esta guía explica qué información preparar para evaluar la integración.

Las Fuentes se incorporan en la **API de Muchi**. El Front muestra sus Resultados;
no tiene un formulario de alta de Tiendas ni importa Inventarios directamente.
La disponibilidad de cada integración debe confirmarse con quienes mantienen
el Servicio.

## Opción 1: Listas de Moxfield

Prepara los enlaces de las Listas que representan el Inventario que deseas
publicar. Deben poder consultarse sin iniciar sesión con tu cuenta.

Incluye esta información:

- Nombre de la Tienda y enlaces de las Listas.
- Cantidad disponible de cada Carta y, cuando corresponda, Edición, Idioma,
  Condición y Acabado.
- Moneda y criterio de Precio: indica si utilizas un Precio de referencia,
  una tasa de Conversión u otra regla. Si hay reglas distintas para Cartas
  normales y foil, descríbelas por separado.
- Enlace de la Tienda o canal donde una persona puede consultar y comprar.
- Frecuencia con la que actualizas las Listas y cómo registras los Productos
  agotados.

Aclara si las Cantidades representan Stock disponible o sólo el contenido de
una Lista de referencia. Una Lista por sí sola no establece el Precio de venta
ni garantiza que sus Cartas estén disponibles.

Comparte esa información al solicitar la integración. El equipo deberá confirmar
cómo la API leerá las Listas y representará los Precios y enlaces de Compra.

## Opción 2: Tu sitio web

Comparte la dirección de tu Tienda y algunos enlaces de Productos que permitan
identificar sus variantes. Si tienes un Catálogo o una API de Stock, incluye
su documentación y un ejemplo de Respuesta sin Credenciales.

La información útil para cada Oferta es:

| Dato | Qué debe representar |
| --- | --- |
| Carta | Nombre y, si está disponible, un Identificador estable. |
| Variante | Edición, Acabado, Condición e Idioma. |
| Precio | Importe de venta y Moneda. |
| Stock | Cantidad disponible o Estado de disponibilidad de esa variante. |
| Enlace | Página donde se puede consultar o comprar la Oferta. |

Si tu sitio expone una API, indica cómo buscar una Carta, recorrer las páginas
del Catálogo y reconocer un Producto agotado. Incluye los límites de Consulta
y cualquier requisito de Autenticación.

Si sólo dispones de páginas de Productos, comparte ejemplos con y sin Stock.
El equipo evaluará si el sitio permite una Consulta fiable y qué Adaptador
necesita la API. No se presupone compatibilidad con todas las plataformas.

## Solicitar la integración

Abre una solicitud en los [Issues del Proyecto](https://github.com/metaliaw/muchi/issues)
con el Nombre de tu Tienda, la opción elegida y la información anterior.
Si no tienes acceso al Repositorio, utiliza el canal de contacto por el que
te compartieron Muchi para coordinar la incorporación.

No incluyas Contraseñas ni Tokens en la solicitud. Si la integración requiere
una Credencial, coordina su entrega por un canal privado y limita sus permisos
a la lectura del Inventario que deseas compartir.

Antes de dar la integración por lista, comprueba con el equipo que los Precios,
las variantes, la disponibilidad y los enlaces correspondan a lo que muestra
tu Tienda. Acuerda también cómo comunicar cambios en tus Listas o en tu sitio.

## Qué verá quien busque tus Cartas

Muchi muestra las Ofertas recibidas de la API, con su Tienda, Precio, Moneda,
Estado de Stock y enlace. La Compra se completa fuera de Muchi, mediante el
enlace de la Oferta. Mantener el Inventario actualizado ayuda a evitar que
aparezcan Cartas agotadas o Precios desactualizados.

[Volver al README](README.md).
