# Más barato que un completo

[English](cheaper-than-a-completo.md) | **Español**

## Una forma simple de sostener las pruebas

Muchi está probando si puede ayudar a encontrar cartas coleccionables a buen precio. La infraestructura cuesta dinero aunque el servicio sea pequeño, y cada aporte ayuda a mantener las pruebas en marcha mientras encontramos qué le sirve a la gente.

La idea de **“más barato que un completo”** pone el costo en una escala cercana. No afirmamos que este trimestre haya costado menos que un completo: primero necesitamos el total real y un precio de referencia local que podamos explicar.

## Período y evidencia disponible

Este documento toma como período el último trimestre calendario cerrado: **1 de julio a 30 de septiembre de 2026**. La consulta se hizo el 5 de octubre de 2026. El reporte de facturación puede incorporar ajustes o uso informado con retraso.

`gcloud` confirmó que el proyecto de Muchi tiene la facturación habilitada y permitió revisar los recursos desplegados. No entregó los cargos del período: el comando de presupuestos encontró deshabilitada la Cloud Billing Budgets API, y esta máquina no tiene instalado `bq`. No habilité APIs ni cambié recursos para obtener los montos.

| Dato del trimestre | Estado |
| --- | --- |
| Cargos de Google Cloud | Pendiente de consultar en Cloud Billing Reports o en una exportación de facturación |
| Créditos y ajustes | Pendiente de consultar en el detalle de facturación |
| Total neto en CLP | Pendiente; no se estimó |
| Equivalente en completos | Pendiente de elegir y publicar un precio de referencia local |

Cloud Billing Reports permite revisar el costo por proyecto, servicio y período; el reporte de tabla de costos ofrece el detalle de la factura. Una exportación a BigQuery permite analizar cargos, créditos y servicios, pero puede generar cargos propios de BigQuery. [Documentación de reportes](https://docs.cloud.google.com/billing/docs/reports) · [Exportación a BigQuery](https://docs.cloud.google.com/billing/docs/how-to/export-data-bigquery)

## Qué infraestructura estaba activa

La consulta de solo lectura a Google Cloud encontró cuatro funciones de segunda generación, una base Firestore, una cola de Cloud Tasks y dos trabajos de Cloud Scheduler en `southamerica-east1`. El código configura un barrido cada cinco minutos y la liberación de órdenes una vez por hora. Estos datos describen recursos; **no permiten deducir el monto de la factura**.

Los cargos pueden venir del almacenamiento y las operaciones de Firestore, las invocaciones y compilaciones de funciones, las tareas, los trabajos programados, los registros y otros servicios del proyecto. El reporte de facturación debe confirmar cuáles generaron cargos netos y cuáles quedaron cubiertos por cuotas gratuitas o créditos.

## Cómo convertir el total en “completos”

Cuando tengamos el costo neto del trimestre, compararemos pesos chilenos con un precio observable de un completo en una fuente y fecha que indiquemos. La cuenta será **costo trimestral en CLP ÷ precio de referencia en CLP**. Como los precios varían por local y ciudad, publicaremos el valor de referencia junto al resultado; no presentaremos la comparación como un precio universal.

## Cómo puede ayudar la comunidad

**Partiría por aportes voluntarios pequeños y patrocinios directos de tiendas.** Son fáciles de explicar y no requieren llenar Muchi de anuncios antes de saber cuánto tráfico tiene.

- **Aporte puntual:** una página de apoyo permite contribuir una vez, con montos voluntarios. Podemos describir cada aporte con la referencia elegida —por ejemplo, “menos de un completo”— y mostrar qué costo ayuda a cubrir.
- **Patrocinio de una tienda:** ofrecer un monto fijo por un período de prueba a tiendas de cartas que quieran apoyar el buscador. El patrocinio debe aparecer rotulado como publicidad y mantenerse separado del orden de resultados, los precios comparados y las recomendaciones.
- **Publicidad automática:** dejarla para después de validar audiencia y experiencia. AdSense revisa los sitios antes de servir anuncios y pide contenido propio, útil y navegación clara. No se debe pedir a los visitantes que hagan clic en anuncios. [Preparar el sitio para AdSense](https://support.google.com/adsense/answer/7299563?hl=es-419) · [Políticas de AdSense](https://support.google.com/adsense/answer/48182?hl=es-419)

Ko-fi es una opción de página de aportes que publica una comisión de servicio de 0% para propinas únicas en el nivel gratuito; PayPal o Stripe pueden sumar sus propias comisiones, que varían por país, moneda y cuenta. Hay que confirmar disponibilidad y costo efectivo para quien recibirá los fondos antes de elegirlo. [Comisiones de Ko-fi](https://help.ko-fi.com/hc/en-us/articles/360002506494-Does-Ko-fi-take-a-fee)

Antes de recibir dinero, definiremos quién lo recibe y cómo se registrará. No anunciaremos beneficios tributarios ni trataremos un aporte como deducible sin revisar el caso aplicable.

## Una prueba breve y transparente

1. Descargar el reporte de Cloud Billing del trimestre cerrado y separar consumo, créditos, impuestos y ajustes.
2. Publicar el costo neto en CLP, la fecha de consulta y el precio usado para convertirlo a completos.
3. Probar durante un período corto un botón de aporte y conversar con algunas tiendas sobre un patrocinio claramente rotulado.
4. Informar cuánto entró, qué comisiones se descontaron y qué parte cubrió de la infraestructura.

La meta inicial no es financiar crecimiento a cualquier costo. Es descubrir si usuarios y tiendas valoran Muchi lo suficiente como para ayudar a pagar las pruebas.
