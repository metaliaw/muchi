[English](entrypoints.md) | **Español**

# Entrypoints de la API

La API declara tres entrypoints de Cloud Functions en [`function.go`](../function.go).
Cada uno es una puerta operativa distinta hacia la misma aplicación compuesta.

- [`ServeAPI`](entrypoints/serve-api.es.md) sirve rutas del contrato público.
  Existe para clientes que necesitan búsquedas, estado, resultados, catálogo
  y salud.
- [`ProcessSearch`](entrypoints/process-search.es.md) atiende un despertar
  privado de Cloud Tasks. Existe para sacar el trabajo lento con tiendas de la
  petición HTTP.
- [`SweepQueue`](queue/sweeper.es.md) repone despertares cuando quedan ítems listos
  después de que los turnos gastaron sus tareas. Existe para recuperar la Cola
  de manera acotada.

Los entrypoints comparten configuración y composición de aplicación, pero
tienen fronteras de confianza distintas: autenticación pública de API, OIDC de
tareas y OIDC del Scheduler. Cada página explica su contrato y razón de ser.
