# Plan: probar la frontera del almacén con un segundo adaptador

[English](provider-agnostic-store-plan.md) | **Español**

Este documento describe trabajo **no realizado**. Es el plan para demostrar que
la frontera descrita en [Arquitectura](arquitectura.md#la-frontera-del-almacén)
aguanta un cambio de proveedor. Hoy esa frontera es una intención bien formada:
existen los puertos, pero un solo adaptador los cumple y nadie ha comprobado
que otro pueda.

## Qué problema resuelve

Muchi API no corre en frío. Levantar el proyecto en una máquina limpia exige un
proyecto de Google Cloud con Firestore y Cloud Tasks, porque los únicos
adaptadores que existen hablan con esos dos servicios. Eso encarece tres cosas
distintas:

- **Contribuir.** Alguien que quiera arreglar un parser de tienda necesita
  credenciales de nube antes de correr el primer test.
- **Probar.** Las pruebas de integración de almacenamiento se saltan salvo que
  haya emulador.
- **Cambiar de proveedor.** Nadie sabe cuánto cuesta hasta que lo intenta.

El objetivo no es abandonar Firestore. Es que la decisión de usarlo siga siendo
una decisión, y no un hecho consumado por omisión.

## Qué construir

### Uno. Un test de contrato compartido

Una única batería de pruebas que reciba un `Vault` y lo someta a las reglas que
el dominio da por ciertas, sin conocer al proveedor:

- Una búsqueda creada se recupera con los ítems que declaró.
- Dos creaciones con la misma llave de idempotencia devuelven la misma búsqueda,
  y una llave repetida con distinto cuerpo entrega conflicto.
- Un reclamo entrega a lo más un ítem por turno, y el arriendo impide que dos
  dueños tomen el mismo.
- Un arriendo vencido vuelve a estar disponible.
- La paginación por cursor no repite ni omite ítems cuando llegan resultados
  entre una página y la siguiente.
- Una oferta guardada con TTL positivo se lee; una vencida no.
- El contador de trabajo pendiente respeta su límite.

Esa batería corre hoy contra el adaptador de Firestore —con emulador— y mañana
contra cualquier otro. Es la pieza que convierte la frontera en algo verificable,
y es la que hay que escribir **primero**: sin ella, el segundo adaptador no
prueba nada.

### Dos. Un adaptador en memoria

Implementa `Vault` y `Dispatcher` sobre estructuras en memoria, con un barrido
propio para la caducidad. Sirve para tres cosas:

- Correr la suite completa sin nube ni emulador.
- Levantar la API local con `MUCHI_STORE=memory` para trabajar en parsers y en
  el contrato HTTP.
- Ser el segundo sujeto del test de contrato, que es lo que realmente demuestra
  que los puertos no filtran supuestos de Firestore.

No se despliega nunca. Un adaptador en memoria que nadie usa se pudre, así que
debe quedar en el camino por defecto del desarrollo local, no en un rincón.

### Tres. La elección del almacén en la composición

`BuildRuntime` elige adaptador según configuración, y sigue siendo el único
lugar del código que nombra un proveedor. El resto del programa ya recibe
puertos y no cambia.

## Lo que hay que resolver por el camino

**La caducidad.** Es el supuesto más profundo y el único que no aparece en
ninguna firma. Firestore borra por política de TTL; el código solo rechaza lo
vencido porque el borrado es eventual. Un adaptador sin TTL nativo debe barrer
por su cuenta, y ese barrido es parte del contrato aunque ninguna interfaz lo
diga. El test de contrato debe exigirlo explícitamente.

**Las transacciones.** El reclamo de ítems, la idempotencia y la cancelación se
apoyan en transacciones. Un almacén sin ellas —una caché, un archivo— no puede
cumplir `Vault` sin volverse incorrecto en silencio. Conviene que el contrato
falle ruidosamente antes que aceptar una implementación floja.

**El despacho.** `Dispatcher` es más fácil: en memoria es una goroutine con una
cola. Lo que no se puede emular es el reintento con backoff de Cloud Tasks, así
que el equivalente local debe ser honesto sobre lo que no hace.

## Orden sugerido

1. Test de contrato contra el adaptador actual, con emulador de Firestore.
2. Adaptador en memoria hasta que pase el mismo contrato.
3. Elección por configuración y suite local sin nube.
4. Recién entonces, si aparece un proveedor candidato real, un tercer adaptador
   ya tiene dónde probarse.

## Lo que este plan no propone

No propone una capa de abstracción nueva ni un ORM. Los puertos ya existen y el
dominio ya está limpio; lo que falta es evidencia, no arquitectura.
