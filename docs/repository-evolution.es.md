[English](repository-evolution.md) · [Español](repository-evolution.es.md)

# Evolución de la Estructura de Repositorios

MUCHI cambió la forma de repartir su código y de abrirlo. La historia tiene
tres etapas; no se fijan fechas porque la documentación disponible registra
las decisiones, pero no el día en que ocurrieron.

## Dos repositorios, uno privado

Al principio, MUCHI vivía en dos repositorios separados. Uno era privado para
proteger la lógica de negocio si alguna vez era necesario hacerlo. Ese fue el
motivo de la privacidad; no encontramos una lista de reglas o módulos que se
quisieran reservar.

## Dos repositorios públicos

Después, ambos repositorios quedaron públicos. La Interfaz y su BFF vivían en
`metaliaw/muchi`; la API, sus Workers y su Persistencia vivían en el otro
repositorio. Cada lado tenía un lugar visible para sus contratos y
responsabilidades. La documentación de la API explicaba su apertura como una
forma de compartir cómo funciona Muchi y construirlo junto a quienes lo usan.

## Un repositorio para el trabajo

El código y la documentación de la API ahora viven aquí junto con la Interfaz
y el BFF. El antiguo repositorio de la API fue deprecado. La API y el BFF
pueden desplegarse como servicios separados, pero su código tiene una sola
fuente en este repositorio.

## Motivos que conocemos

- **Proteger la lógica de negocio:** ese fue el motivo para mantener uno de
  los repositorios privado al principio, según el recuerdo del proyecto. Este
  motivo queda registrado a partir de la aclaración del usuario.
- **Compartir y construir en comunidad:** la documentación del proyecto
  registró este motivo para hacer público el código.
- **Separar responsabilidades:** el diseño anterior asignaba la experiencia y
  el BFF a este repositorio, y la búsqueda, los Workers y la Persistencia al
  repositorio de la API. El código está consolidado aquí; los roles de
  ejecución pueden seguir separados.
- **Trabajar desde un repositorio:** se deprecó el repositorio separado para
  que el trabajo futuro, los contratos y la documentación tengan una sola
  fuente aquí.

La revisión buscó referencias sobre privacidad, visibilidad, lógica de negocio
y repositorios en la documentación disponible. Las menciones a funciones
«privadas» y a `chaos/` describen controles de acceso de servicios y datos
sensibles, no la decisión histórica de hacer privado el repositorio.

La estructura actual se documenta en:

- [Integración con la API](api/README.es.md) — cliente, configuración y frontera.
- [Arquitectura de Muchi](architecture.es.md) — recorrido, seguridad y despliegue.
- [Hallazgos en los Buscadores](search-findings.es.md) — decisiones en la frontera.

## Qué cambió

| Etapa | Código visible | Fuente de trabajo |
| --- | --- | --- |
| Dos repositorios, uno privado | Parcial | Repartida |
| Dos repositorios públicos | Completo en ambos | Repartida entre los dos |
| Un repositorio activo | Consolidado en `muchi` | Este repositorio |

La historia avanza de una frontera privada a una pública, y luego a una sola
fuente de código, documentación y trabajo futuro. La API y el BFF todavía
pueden correr como servicios separados.
