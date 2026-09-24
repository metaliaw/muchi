[English](serve-api.md) | **Español**

# ServeAPI Expone el Contrato de Búsqueda

`ServeAPI` es el entrypoint público de Cloud Functions. Construye el handler
HTTP una vez por instancia y sirve el contrato OpenAPI: autenticación, salud,
creación y lectura de búsquedas, cancelación, comprobaciones de stock y
consultas de catálogo.

Existe como frontera de red estable para el Front y las integraciones. Las
rutas de negocio requieren el Bearer Token configurado; la ruta básica de salud
permanece pública para comprobaciones del servicio. El handler valida las
peticiones de transporte y delega reglas de negocio al servicio de aplicación.

El entrypoint se registra en [`function.go`](../../function.go) y se compone
en [`internal/httpapi`](../../internal/httpapi).
