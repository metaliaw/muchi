[English](muchi.md) · **Español**

# Muchi Entrega la Voz Curada de la Interfaz

`GET /api/muchi` entrega frases, saludos, temas de ayuda y respuestas según
contexto desde el catálogo común. Una sola ruta permite que el cliente muestre
una voz consistente sin copiar los datos fuente al paquete de Vue.

La respuesta agrupa frases por uso y conserva el estado de cada una. No decide
cuándo hablar; la Interfaz elige entre las líneas disponibles.

La ruta está implementada por [`read_muchi`](../../server/main.py).
