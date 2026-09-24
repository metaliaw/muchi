[English](sealed-product.md) · [Español](sealed-product.es.md)

# Producto Sellado

Muchi Buscaba Cartas sueltas. Una Caja de Sobres Parecía lo mismo con otro
Precio, y no lo es: se Nombra distinto, se Agrupa distinto, y su Foto Viene de
otro Lado.

La API Aprendió `kind=sealed`. Este Documento Cuenta qué Cambió de este Lado de
la Frontera, y sobre todo **qué se Rompió al Probarlo** — porque casi nada de lo
que se Rompió era del Sellado. Estaba ahí antes, tapado por el Hecho de que una
Carta suelta Tiene un Catálogo al que Preguntarle y una Caja no.

Los Defectos del otro Lado —los del Contrato y las Fuentes— Viven en
`docs/sealed-product.es.md` del
[Repositorio muchi-api](https://github.com/cangrejometralleta/muchi-api). Cada
Repositorio Guarda los suyos, por la misma Razón que en los
[Hallazgos en los Buscadores](search-findings.es.md): una Copia del Documento
Ajeno Envejece mal.

## El Catálogo se Elige, no se Deduce

El Formulario Lleva un `fieldset` «Qué Buscar» con dos Opciones, hermano del que
ya Existía para la Coincidencia:

```
Juego:  [ Magic ▾ ]
Qué Buscar:  ( • Cartas sueltas )  ( ○ Producto sellado )
```

Una Búsqueda es **entera** de Cartas o **entera** de Cajas. Mezclarlas Obligaría
a Decidir Línea por Línea qué es cada Cosa, y «Caja de Sobres» contra «Sol Ring»
no se Distingue por el Texto.

El `kind` Viaja en la URL junto al `match` —una Búsqueda de Cajas Retomada por su
Enlace no Vuelve como una de Cartas— y se Recuerda en `localStorage` junto al
Juego.

### Sellado no Fuerza `includes`

Parecía obvio que sí: ninguna Tienda Titula una Caja igual que la otra, así que
la Coincidencia ancha es la única que Sirve.

Pero el `wide` de este Formulario no solo Ensancha: **Recorta la Lista a su
primera Línea**, porque Mirar una Familia de Cartas es de a una. Una Lista de
Cajas con Cantidades es tan legítima como una de Cartas, y forzarlo habría tirado
la Línea 2 en adelante.

Además es innecesario. La API ya Ensancha sola toda Pregunta sellada, y la Agrupa
por su Llave sin Mirar el Modo. Así que en Sellado el `fieldset` de Coincidencia
**se Esconde**: Habla de Impresiones y Derivados, dos Cosas que una Caja sin Abrir
no Tiene.

## El Campo Llega Escrito

Cada Combinación de Juego y Catálogo Trae un Valor por Defecto, y los seis están
**medidos contra las Tiendas**, no Inventados:

| | Cartas | Sellado | Ofertas medidas |
| --- | --- | --- | --- |
| Magic | `Sol Ring` | `Play Booster` | 32 |
| Pokémon | `Pikachu` | `Prismatic Evolutions Booster Bundle` | 3 |
| Yu-Gi-Oh | `Dark Magician` | `Booster Box` | 6 |

La Lección que Dejaron las Mediciones: **el Nombre corto Encuentra más que el
largo.** `Maze of Millennia Booster Box` se Pasa del Tiempo de Espera; `Booster
Box` Trae seis. Las Tiendas chilenas Escriben el Set adelante y a su manera, así
que el Título que Viene impreso en la Caja de Fábrica no Encuentra nada.

El Ejemplo Sigue al Juego y al Catálogo **mientras nadie Haya escrito lo suyo**.
Un Texto propio Manda: cambiar de Juego no le Borra la Lista a nadie.

## Los Números Sobreviven

El Parser de Listas Recorta el Número suelto del final, porque en una Carta es su
Número de Colección: `1 Sol Ring (LTC) 344 *F*` Pide un Sol Ring.

En una Caja ese Número es parte del Nombre.

```
antes:  'Set de Batalla 2024'  →  Order(1, 'Set de Batalla')
```

`strip_decorations` Ahora Parte sus Patrones en dos. Las **Marcas** —foil,
`#!Commander`, Corchetes— se Sacan siempre. La **Impresión** —la Edición entre
Paréntesis y el Número— solo en Cartas sueltas.

## La Caja no Tiene Catálogo

`CardArt` le Pide la Imagen al Catálogo del Juego. Para una Caja no Hay a quién
Preguntarle: `/cards/metadata` Conoce Cartas.

En Sellado el Panel **no Consulta nada** y Usa la Foto que Publicó la Tienda.
Preguntar igual Habría Dejado el Panel en «Buscando la Imagen…» para siempre.

Por la misma Razón el Buscador Asistido se Esconde: `/cards/autocomplete` tampoco
Toma `kind`, y un Buscador que nunca Encuentra es peor que ninguno.

## La Imagen se Perdía en Cuatro Capas

Este es el Hallazgo que más Costó Encontrar, y el que más Enseña.

La API **sí** Manda la Foto de cada Oferta. Medido: 31 de 32 en Magic, 3 de 3 en
Pokémon. Pero no Llegaba al Navegador, y se Perdía cuatro veces seguidas:

| Capa | Qué pasaba |
| --- | --- |
| `muchi/api/client.py` | `build_offer` Leía veinte Campos del JSON y `image` no era ninguno |
| `muchi/mtg/search.py` | `SearchOffer` no Tenía dónde Guardarla |
| `server/presenter.py` | La Fila que Viaja al Navegador tampoco la Llevaba |
| `web/src/components/OfferList.vue` | Y el Front la Buscaba en `offer.metadata.image` — un Lugar donde la API nunca la Pone |

Ninguna de las cuatro se Había notado, y la Razón es la misma que Vuelve
interesante al Sellado entero: **para una Carta suelta nadie Necesita ese Campo.**
`CardArt` le Pide la Imagen al Catálogo por su Cuenta. El Hueco Existía desde
siempre y solo se Vio cuando Apareció algo que no Tiene Catálogo.

`image` Quedó como último Campo de `SearchOffer` a propósito: hay Pruebas que
Arman Ofertas por Posición.

## Decir «No Hay» es Afirmar lo que no se Sabe

Una Pantalla Mostraba esto, junto:

```
Bloomburrow Play Booster: no se pudo consultar lacripta.cl; faltan sus Ofertas.
No hay Ofertas para mostrar.
```

La primera Línea es un Fallo. La segunda es una Ausencia. Y una Ausencia
calculada sobre Fuentes que no Contestaron no es una Ausencia: es un Hueco.

El Contrato ya lo Decía —una Lista de `faults` no vacía Significa Respuesta
incompleta— y la Interfaz lo Mostraba al lado de su Contradicción.

Hoy, cuando Hay Avisos de Nivel `warning`, el Texto Cambia:

> Ninguna Oferta llegó, y algunas Fuentes no contestaron. Lo que falta puede
> existir igual: reintenta en un rato.

Sale de los `notices` que ya Llegaban, sin Tocar el Contrato ni el Proxy.

## El Campo que Desaparecía

El Polling Devolvía **502** en cada Ciclo, con «La Respuesta no cumple el Contrato
de la API» y ninguna Pista de cuál Campo.

Eran dos Campos que la API Declara obligatorios y Manda ausentes: `sequence`, que
Lleva `omitempty` y Desaparece cuando Vale cero, y `offers`, que Llega `null` en
vez de Lista vacía cuando un Item no Encontró nada. Los dos son Estados normales
de un Item que Corre o que no se Halló.

`build_results` Ahora los Lee con Tolerancia. Y `parse_reply` **Nombra el Campo**:

```
La Respuesta no cumple el Contrato de la API: falta el Campo «sequence».
```

Eso era lo único que Servía para Arreglarlo, y el Mensaje se lo Callaba.

> El `omitempty` se Corrigió también del lado que Mentía. Está contado en el
> Documento gemelo de muchi-api.

## Qué Queda Abierto

- **Ninguna Búsqueda sellada Real desde la Interfaz.** Todo lo de acá se Verificó
  consultando la API directo. Falta Abrir la UI, Lanzar una Lista de Cajas y Ver
  la Foto en el Panel lateral.
- **Tres de seis Ofertas de Yu-Gi-Oh Llegan sin Foto.** Esas Tiendas no la
  Publican, y no Hay de dónde Sacarla sin Inventarla. El Panel ya Maneja el Caso:
  sin `image`, no Dibuja nada.
- **El Carrito nunca se Probó con Cajas.** Debería Funcionar igual —las
  Cantidades son Cantidades— pero es un Supuesto, no una Medición.
