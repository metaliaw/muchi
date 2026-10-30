[English](README.md) · [Español](README.es.md)

# Integración con la API

Este repositorio consume la API de Muchi mediante un cliente del lado servidor.
El Navegador llama al BFF; las credenciales y las peticiones externas se quedan
en el servidor.

## Dónde Leer

| Tema | Documento o código |
| --- | --- |
| Implementación de la API: Fuentes, Workers y Tiendas | [`backend/`](../../backend/) |
| Cliente de API, configuración y formas de respuesta | [`bff/muchi/api/`](../../bff/muchi/api/) |
| Rutas públicas del Navegador y su frontera | [Entrypoints del BFF](../bff/README.es.md) |
| Recorrido, seguridad y despliegue | [Arquitectura de Muchi](../architecture.es.md) |
| Presentación y lectura de respuestas | [`bff/server/presenter.py`](../../bff/server/presenter.py) |
| URL y token de ejecución | [`bff/.env.example`](../../bff/.env.example) |

La implementación de la API y su integración pertenecen a este repositorio.
La API y el BFF pueden correr como servicios separados, pero su código y su
documentación tienen aquí una sola fuente. La
[nota sobre la evolución de los repositorios](../repository-evolution.es.md)
registra el paso desde la estructura dividida.
