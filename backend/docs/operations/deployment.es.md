# Despliegue con Gatos

[English](deployment.md) | **Español**

## Resumen

`deploy.sh` y `deploy.cmd` ejecutan `gcloud` directamente con tu sesión de
`gcloud auth login`. Usan el proyecto activo de Google Cloud CLI o `--project`.
No requieren Go local para desplegar: GCP compila las funciones desde el código
fuente.

```sh
./deploy.sh
```

```bat
deploy.cmd
```

No hay despliegue automático desde GitHub. Cada despliegue sale de una máquina,
con la sesión y los [permisos](#permisos) de quien lo ejecuta.

## Los Cuatro Componentes

En Unix también puedes desplegar cada componente por separado. `deploy.sh` los
orquesta en este mismo orden:

```sh
./deploy-infra.sh
./deploy-worker.sh
./deploy-sweeper.sh
./deploy-api.sh
```

El orden no es arbitrario. La API y el barredor descubren la URL del worker ya
desplegado: requieren que exista, pero no vuelven a desplegarlo.

Un cambio de código sólo necesita los tres últimos. La infraestructura casi nunca
cambia, y separarla también reduce los permisos del despliegue diario.

## Qué Hace cada Script

`deploy-infra.sh` habilita los servicios, reutiliza o crea cuentas, Firestore y
la cola, y solicita los TTL de las cinco colecciones. Un Firestore existente
conserva su ubicación.

`deploy-worker.sh` despliega el worker privado `ProcessSearch` y concede su
invocación a Cloud Tasks mediante OIDC.

`deploy-sweeper.sh` deja el barredor y su despertador de Cloud Scheduler, cada
cinco minutos.

`deploy-api.sh` despliega `ServeAPI`, pública en IAM y protegida por el token de
la aplicación, con la URL real del worker. No se inicia un worker residente
en GCP.

## Permisos

Quien despliegue necesita estos roles en el proyecto:

```text
roles/serviceusage.serviceUsageAdmin    habilita las APIs
roles/iam.serviceAccountAdmin           crea las cuentas de ejecución
roles/resourcemanager.projectIamAdmin   les concede sus roles
roles/iam.serviceAccountUser            despliega en nombre de ellas
roles/cloudfunctions.admin              despliega las funciones
roles/run.admin                         las revisiones detrás de ellas
roles/storage.admin                     el código fuente que sube el build
roles/cloudtasks.admin                  la cola de búsquedas
roles/cloudscheduler.admin              el despertador del barredor
roles/secretmanager.admin               lee y versiona el token
roles/datastore.owner                   Firestore, sus índices y sus TTL
```

Concede uno así, repitiendo por rol:

```sh
gcloud projects add-iam-policy-binding mi-proyecto \
  --member='user:quien@ejemplo.cl' --role=roles/cloudfunctions.admin
```

Los tres primeros sólo los necesita `deploy-infra.sh`. Quien despliegue
únicamente worker, barredor y API puede prescindir de ellos, y así el despliegue
diario no puede reescribir los permisos del proyecto por accidente.

Las cuentas de servicio que crea `deploy-infra.sh` son otra cosa: gobiernan la
ejecución en GCP, no el despliegue. El [diagrama de arquitectura](../architecture.es.md)
muestra dónde vive cada identidad.

Esta lista se dedujo de lo que hacen los scripts, no de una sesión real con
permisos mínimos. Si estrenas una cuenta nueva y falta o sobra un rol, corrige
aquí.

El proyecto debe existir y tener facturación activa.

## Opciones

Los valores compartidos están en `config/deploy.env`: las funciones
`muchi-serve-api` y `muchi-process-search`, región `southamerica-east1`, cola
`muchi-searches`, cuentas de servicio y límites. Puedes seleccionar otro
proyecto, región o referencia de Secret Manager:

```sh
./deploy.sh --project mi-proyecto --region southamerica-east1 --token-secret muchi-api-token
```

Agrega `--dry-run` para revisar los comandos sin cambios en GCP.

## El Token

Ambos servicios reciben la referencia viva `muchi-api-token:latest`: cada
instancia nueva lee la última versión habilitada, así que rotar el secreto no
exige redesplegar. El despliegue verifica que esa última versión exista y esté
habilitada, y avisa si pasas un número distinto.

El secreto debe existir antes del primer despliegue: sin él los scripts fallan al
verificar que su última versión esté habilitada, y el error aparece ya empezado
el despliegue.

Para crearlo por primera vez o agregar una versión, usa `--token-file`
con la ruta absoluta de un archivo privado fuera del repositorio, sin salto de
línea final. El token no se imprime ni se pasa como valor en argumentos. Los
scripts no cargan `.env`.

Rotar el secreto no basta por sí solo: el
[incidente de despliegue, revisiones y secretos](deployment-revisions-secrets.es.md)
explica por qué una rotación correcta también debe mover tráfico y actualizar a
todos los consumidores.

## Lo que Viaja y sus Límites

Se incluyen las tiendas de `config/stores.yaml` en el código desplegado; el
runtime Go las lee desde `serverless_function_source_code/config/stores.yaml`.

Los límites predeterminados son 512 MiB, timeout de 1800 segundos, concurrencia 1
y hasta 5 instancias por función. La cola admite 1 envío por segundo y 2 tareas
concurrentes. Las tareas HTTP tienen un plazo de 1800 segundos
(`MUCHI_TASK_DEADLINE_SECONDS`), para completar búsquedas Jumpseller con
paginación extensa. En local, usa el flujo asíncrono para estas búsquedas.

## La Salida

Incluye gatos y estados, junto con el progreso y los errores nativos de gcloud.
Un error detiene el script; los recursos ya creados se conservan para reintentar.
Al terminar se imprime la URL de la API y su ruta `/v1/health`.
