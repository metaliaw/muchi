[English](search-findings-en.md) · [Español](search-findings.md)

# Hallazgos en los Buscadores

Muchi Mostraba una Lista de Ofertas y Coronaba la más barata. Eso Funcionó
mientras cada Búsqueda Traía **una Carta y sus Impresiones**, que es lo único que
la API Devolvía.

El Día que la API Aprendió `match=includes`, una Búsqueda Empezó a Traer **Cartas
distintas**: buscar «Kuriboh» Encuentra también Winged Kuriboh, Linkuriboh y
Token: Kuriboh. Ahí se Cayeron cuatro Supuestos que nadie Había Escrito, porque
hasta ese Día eran ciertos.

El quinto Hallazgo no Viene de ahí. Salió al Pasar por al lado.

Los Defectos del otro Lado de la Frontera —los de las Fuentes que Buscan— Viven
en `docs/search-findings.md` del
[Repositorio muchi-api](https://github.com/cangrejometralleta/muchi-api).
Cada Repositorio Guarda los suyos: una Copia del Documento Ajeno Envejecería igual
que Envejeció la Copia del Contrato, que es justamente F5.

## Dónde Decide Cada Cosa

`server/presenter.py` Decide y `web/src/search.js` es su **Gemelo** del otro Lado
de la Frontera. El BFF Manda una Página por Ciclo y el Front Acumula, así que la
misma Decisión está Escrita dos veces. Cuatro de estos cinco Hallazgos Tocaron a
los dos Gemelos.

```
La API Responde          card_key · faults · ofertas
        │
        ▼
server/presenter.py      agrupa por Tipo de Carta, corona, arma el Carrito
        │  una Página por Ciclo
        ▼
web/src/search.js        acumula las Páginas y rehace el Resumen
        │
        ▼
components/OfferList     dibuja, no decide
```

## Los Hallazgos

| ID | Hallazgo | Estado |
| --- | --- | --- |
| F1 | Una sola «más barata» para toda la Página | Cerrado `a1bc9d8` |
| F2 | El Carrito Compraba la Carta equivocada | Cerrado `a1bc9d8` |
| F3 | Agrupar por Texto Parte una Carta en varias | Cerrado `57e571e` |
| F4 | Nadie Sabía que Faltaba una Tienda | Cerrado `b972a65` |
| F5 | La Copia del Contrato Llevaba 130 Líneas de Atraso | Cerrado `03a4d92` |

### F1. Una sola «más barata» para toda la Página

`order_offers` lo Decía en su propia Docstring: *«Abre por Precio, barata primero,
mezclando todas las Cartas»*. Y `pick_cheapest` Coronaba **una sola** Oferta para
toda la Respuesta.

```
 100 CLP  Linkuriboh        🐾 el mas barato
 300 CLP  Winged Kuriboh
 900 CLP  Kuriboh           ← lo que se pidió
```

Comparar el Precio de un Linkuriboh con el de un Kuriboh no Dice nada: son Cartas
distintas. Cada Tipo de Carta Corona la suya, y la Vista los Separa con
Encabezado cuando hay más de uno.

**El Tipo de Carta Depende del Modo.** En `exact` el Tipo es la Carta Pedida,
porque sus Impresiones son la misma Carta y Compiten entre ellas. En `includes`
cada Título es un Tipo. Es una Regla dicha en Lenguaje de Dominio, no dos Reglas.

### F2. El Carrito Compraba la Carta equivocada

`build_cart` Agrupaba por el Nombre Pedido y Dejaba Entrar toda Oferta del Ítem.
Con Derivados, el Optimizador Compraba el Derivado porque Salía más barato.

```
sin el arreglo   total 4700   1× Winged Kuriboh LV9   ✗
con el arreglo   total 5000   1× Kuriboh              ✓
```

Los Derivados son para **Mirar**, no para Comprar de a tres. En `includes` el
Carrito Vuelve a la Carta Pedida, Filtrando con el Gemelo de `offer.MatchesCard`
que Vive en `names_same_card`.

⚠️ Ése es un tercer Gemelo, y el único que Cruza la Frontera: Repite una Regla que
la API ya Aplica del otro Lado. Está ahí porque el Carrito Necesita Angostar lo que
una Búsqueda ancha ya Trajo.

### F3. Agrupar por Texto Parte una Carta en varias

`read_card_type` Bajaba el Título a minúsculas y lo Usaba como Carta. El Front
Estaba Resolviendo con Texto una Pregunta que sólo la API Puede Responder, porque
es la que Conoce las Reglas de Calce.

```
Winged Kuriboh                                     ┐
LDS3-EN100 “Winged Kuriboh” Common Effect Monster  ├→ tres Grupos
Winged Kuriboh (PUR)                               ┘
```

Cada Fuente Escribe el Título a su Manera, y agrupar por ese Texto Partía una
Carta en tantos Grupos como Formas de Escribirla Hubiera. Cada Grupo con su propia
«más barata», que es como Decir ninguna.

La API ahora Manda `card_key` y Responde la Pregunta una vez. El Front la Lee y
**Cae al Título cuando Viene vacía**, que es lo que Responde una Versión anterior
de la API: Desplegar las dos Puntas al mismo Tiempo no Siempre se Puede.

### F4. Nadie Sabía que Faltaba una Tienda

La API ya Decía qué Fuente se Había Caído y el Front no lo Leía. Una Búsqueda con
una Tienda menos Llegaba `found` con sus Ofertas y se Veía completa.

```
⚠ Kuriboh: no se pudo consultar v3.netdecker.cl; faltan sus Ofertas.
```

**La Vista no Cambió.** `notices` ya Existía y ya se Dibujaba con su Estilo de
Aviso; lo que Faltaba era Llenarlo. Ése es el mejor Resultado posible de un
Cambio así.

El Front se Queda con el **Nombre** de la Fuente y Tira la **Razón**. La Razón
Trae el Cuerpo de la Respuesta, y una Tienda en Mantención Contesta una Página de
HTML. Eso es para el Log; quien Busca sólo Necesita Saber que este Precio se
Comparó con una Tienda menos. La Prueba Exige que `<!doctype` no Aparezca nunca en
el Texto visible.

### F5. La Copia del Contrato Llevaba 130 Líneas de Atraso

`docs/api/openapi.yaml` Declaraba en su primera Línea que era una Copia y que se
Sincronizaba desde `muchi-api`. Le Faltaban `/supported-games`, la Paginación de
Resultados, `match`, `card_key`, `faults` y `SourceFault`, y Seguía Declarando
`stores_only` como requerida cuando ya no Existía.

Una Copia que Envejece en Silencio es peor que no Tenerla: quien la Lee Cree que
está Mirando el Contrato. Primero se Copió entera desde la Fuente. Después
`muchi-api` se Hizo público y la Copia se Borró junto con la de Bruno: ahora se
Enlaza el Contrato donde Vive, que es la única Sincronización que no se Olvida.

Este Documento Existe por la misma Razón, al revés: los Hallazgos de la API no se
Copian acá.

## Lo que Queda Abierto

1. **El Front no Tiene Runner de JS.** `web/src/search.js` es el Gemelo de
   `server/presenter.py` y sólo el Lado Python está Probado. Nada Impide que los
   Gemelos Deriven — en la API, tres Copias de la misma Regla de Calce Derivaron
   exactamente así, y ninguna Prueba lo Notó.
2. **`deploy.sh` no Corre las Pruebas.** Ni `pytest` ni el Build del Front. Hay
   que Llamarlos a Mano antes de Desplegar.
3. **Ninguna Vista se Miró en un Navegador.** El Selector de Modo, los
   Encabezados de Grupo y el Aviso de Fuente Caída se Verificaron por API contra
   los dos Servicios Desplegados y por Pruebas, no Mirando la Página.
4. **El Carrito Repite una Regla de la API** (`names_same_card`). Si la Regla de
   Calce Cambia allá y no acá, el Carrito Empieza a Comprar distinto sin que nada
   lo Diga.

## Qué Dejó Esta Ronda como Método

- **Los Gemelos se Cambian juntos o se Separan.** Cada Hallazgo que Tocó
  `presenter.py` Tocó también `search.js`, en la misma Vuelta.
- **Probar por Mutación.** Cada Arreglo Tiene un Caso que Falla si se Revierte:
  sin la Barata por Tipo Gana el Linkuriboh, y sin el Filtro del Carrito se
  Compra el Linkuriboh.
- **Desplegar el Consumidor antes que el Contrato.** Al Retirar `stores_only`, el
  Front Dejó de Mandarla y se Desplegó **primero**: la API Usa
  `DisallowUnknownFields` y el Orden inverso Habría Devuelto `400` a cada
  Búsqueda.
