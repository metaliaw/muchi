# Por qué la Re-verificación de Stock es Opcional

Septiembre 2026 · Rama `deploy.next`

## La Cadena de Carga

Una Búsqueda de 50 Cartas consulta las Fuentes una vez por Carta. Hasta ahí,
la Carga crece con la Lista y nadie se sorprende.

La Re-verificación (`verify_stock`) agrega una **segunda Vuelta**: por cada
Oferta barata encontrada, el Servicio visita la Tienda de nuevo para confirmar
que el Stock sigue en pie. La Cadena queda así:

```
1 Búsqueda = N Consultas de Precio + M Consultas de Stock
             (una por Carta)      (una por Oferta candidata)
```

M no tiene Tope proporcional a N: una Carta común con veinte Tiendas baratas
genera veinte Visitas extra. El Mazo de Commander —cien Cartas, Tierras
repetidas— es exactamente el Caso que más Re-verifica.

## El Efecto en el Sistema

- **La Búsqueda tarda más en Cerrar.** El Usuario mira una Barra de Progreso
  que no avanza mientras el Servicio consulta Stocks que tal vez ya no
  importan: el Carrito solo toma la más barata por Tienda.
- **Las Tiendas reciben el Doble de Tráfico.** Somos Invitados en sus
  Catálogos; duplicar Visitas por cortesía de una Certeza efímera es una
  Deuda que alguien Notará.
- **La Certeza caduca al instante.** El Stock confirmado a las 21:00 puede
  agotarse a las 21:01. La Re-verificación compra minutos de Certeza a precio
  de minutos de Espera.

## El Incidente que lo Hizo Visible

El Front caía en *Reintentar Envío* sin Resultados. La Cadena del Fallo:

1. El BFF tardaba de más (la Re-verificación infla cada Ciclo) y cortaba en 5xx.
2. El Cliente marcaba el Fallo como reintentable y conservaba el Envío.
3. Sin `searchId`, el Polling nunca arrancaba: solo quedaba el Botón.

El Reintento en sí no es el Villano — es el Termómetro. Cada Reintento reenvía
la Búsqueda entera, y si la Carga era la Causa, la Carga crece. Un Sistema
sobrecargado que Reintenta es un Sistema que se Pisa solo.

## La Decisión

`MUCHI_VERIFY_STOCK` nace **apagada**. El Carrito confía en el Precio visto en
la primera Pasada, como lo hace cualquier Comparador. Y coherente con eso:
cuando la Flag está apagada, ninguna Oferta declara Stock — sin Certeza no se
muestra Badge, ni "En Stock", ni "No confirmado".

Encenderla es una línea en el Entorno (`MUCHI_VERIFY_STOCK=1`), sin Deploy.
Se reserva para el Día en que la Certeza pese más que la Carga — por ejemplo,
una Venta nocturna masiva donde el Stock se mueve por Minuto.

## Lo que Queda

- El Reintento conserva su Clave de Idempotencia: reenviar no duplica
  Búsquedas. Eso no se toca.
- El Error de Red ahora dice lo que Es ("La Consulta no llegó al Servicio")
  en vez de un `TypeError` mudo.
- El Front siente qué Cambió en cada Ciclo (`stateChanges`): el camino a
  Deltas parciales queda abierto sin haberle pedido nada nuevo a la Red.
