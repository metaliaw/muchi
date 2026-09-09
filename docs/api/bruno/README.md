# Colecciones de Bruno

Copia de referencia. La Fuente oficial es `bruno/` del
[Repositorio privado muchi-api](https://github.com/cangrejometralleta/muchi-api);
los Cambios se hacen allá y se sincronizan aquí, junto con
[openapi.yaml](openapi.yaml). Esta Copia no se mantiene por su cuenta.

Sirven para pegarle a la API **sin pasar por el BFF**, que es como se separa
un Fallo del Front de uno del Servicio. El BFF guarda el Token y el Navegador
nunca lo ve; acá lo pone quien consulta.

Abre `docs/api/bruno/Muchi API` en Bruno, o córrelas desde una Terminal:

```bash
cd docs/api/bruno/Muchi\ API
npx @usebruno/cli run --env Local -r
```

`Local` espera la API en `http://127.0.0.1:8081` con el Token de Desarrollo
de `compose.yaml`. `Production` deja el Token vacío a propósito: es una
Variable Secreta, Bruno la pide y no la escribe en Disco.

`Create Search` guarda el Identificador devuelto en el Entorno, así que corre
esa antes que Estado, Resultados o Cancelar.
