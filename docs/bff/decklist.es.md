[English](decklist.md) · **Español**

# Decklist Anticipa la Lectura Antes de Buscar

`POST /api/decklist` interpreta el texto pegado y devuelve las cantidades
reconocidas y las líneas ignoradas. La Interfaz puede mostrar qué incluiría
la búsqueda antes de iniciar trabajo en el backend.

Esta vista previa usa el mismo analizador de producción, incluso para
productos sellados. No crea una búsqueda, no reserva trabajo ni consulta
tiendas. Las líneas inválidas quedan visibles para la Persona en vez de
convertirse silenciosamente en otra lista.

La ruta está implementada por [`read_decklist`](../../server/main.py).
