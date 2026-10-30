[English](process-search.md) | **Español**

# ProcessSearch Convierte un Despertar en una Unidad Reclamada

`ProcessSearch` es el destino HTTP privado de Cloud Tasks. Cada invocación pide
al worker reclamar el siguiente ítem disponible, consultar sus fuentes
configuradas y persistir ofertas y avance.

La tarea no lleva identificador de ítem: así workers competidores pueden
reclamar transaccionalmente y la recuperación de leases permite avanzar tras
un turno fallido. `X-CloudTasks-TaskName` se usa como dueño del reclamo. Si no
hay ítem, el turno vacío es normal y responde `204`; otros fallos responden
`500` para que Cloud Tasks reintente. La función requiere invocación OIDC
autenticada.

El entrypoint se registra en [`function.go`](../../function.go); el reclamo y
la búsqueda se componen desde `internal/application` e `internal/search`.
