@echo off
rem Deploys Muchi with the Active gcloud Session.
setlocal DisableDelayedExpansion
chcp 65001 >nul
pushd "%~dp0" || exit /b 1
for /f "usebackq eol=# tokens=1,* delims==" %%A in ("config\deploy.env") do set "%%A=%%B"
set "PROJECT="
set "TOKEN_FILE="
set "SECRET_NAME="
set "SECRET_VERSION="
set "DRY_RUN=0"
set "STEP=Configuración"
set "DEPLOY_READ=%TEMP%\muchi-deploy-%RANDOM%-%RANDOM%.txt"
:Parse
if "%~1"=="" goto Ready
if "%~1"=="--dry-run" (
  set "DRY_RUN=1"
  shift
  goto Parse
)
if "%~1"=="--help" goto Help
if "%~1"=="-h" goto Help
if "%~2"=="" goto Failed
if "%~1"=="--project" (set "PROJECT=%~2") else if "%~1"=="--region" (set "REGION=%~2") else if "%~1"=="--token-secret" (set "MUCHI_API_TOKEN=%~2") else if "%~1"=="--token-file" (set "TOKEN_FILE=%~2") else goto Failed
shift
shift
goto Parse
:Help
echo 🐱 deploy.cmd [--project ID] [--region REGION]
echo   [--token-secret NOMBRE:VERSION] [--token-file ARCHIVO] [--dry-run]
goto Done
:Ready
where gcloud >nul 2>nul || goto Failed
if not defined PROJECT call :ReadProject || goto Failed
if not defined PROJECT goto Failed
if "%PROJECT%"=="(unset)" goto Failed
for /f "tokens=1,2 delims=:" %%A in ("%MUCHI_API_TOKEN%") do (
  set "SECRET_NAME=%%A"
  set "SECRET_VERSION=%%B"
)
if not defined SECRET_VERSION goto Failed
if defined TOKEN_FILE if not exist "%TOKEN_FILE%" goto Failed
set "API_EMAIL=%API_ACCOUNT%@%PROJECT%.iam.gserviceaccount.com"
set "WORKER_EMAIL=%WORKER_ACCOUNT%@%PROJECT%.iam.gserviceaccount.com"
set "TASK_EMAIL=%TASK_ACCOUNT%@%PROJECT%.iam.gserviceaccount.com"
set "BUILD_EMAIL=%BUILD_ACCOUNT%@%PROJECT%.iam.gserviceaccount.com"
echo 🐱 Muchi en GCP
echo Proyecto: %PROJECT%
echo Región: %REGION%
if "%DRY_RUN%"=="1" echo ⚠️ Vista Previa: GCP sin Cambios
call :Services || goto Failed
call :Accounts || goto Failed
call :Database || goto Failed
call :Worker || goto Failed
call :API || goto Failed
if "%DRY_RUN%"=="1" (
  echo ✅ Vista Previa Completa
) else (
  echo 😺 API y Worker Desplegados
  echo API: %API_URL%
  echo Salud: %API_URL%/v1/health
)
goto Done

:ReadProject
call gcloud config get-value project >"%DEPLOY_READ%"
if errorlevel 1 exit /b 1
for /f "usebackq delims=" %%V in ("%DEPLOY_READ%") do set "PROJECT=%%V"
exit /b 0

:Cloud
if "%DRY_RUN%"=="1" (
  echo gcloud %* --project=%PROJECT% --quiet 1>&2
  exit /b 0
)
if defined PROJECT (
  call gcloud %* --project="%PROJECT%" --quiet
) else (
  call gcloud %* --quiet
)
exit /b %ERRORLEVEL%

:ReadCloud
set "%~1="
call :Cloud %2 %3 %4 %5 %6 %7 %8 %9 >"%DEPLOY_READ%"
if errorlevel 1 exit /b 1
for /f "usebackq delims=" %%V in ("%DEPLOY_READ%") do set "%~1=%%V"
del /q "%DEPLOY_READ%" >nul 2>nul
exit /b 0

:Services
set "STEP=Servicios y Secreto"
echo 🐱 %STEP%
call :Cloud services enable cloudfunctions.googleapis.com run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com firestore.googleapis.com cloudtasks.googleapis.com secretmanager.googleapis.com iam.googleapis.com iamcredentials.googleapis.com || exit /b 1
if defined TOKEN_FILE call :UploadToken || exit /b 1
if "%DRY_RUN%"=="1" set "SECRET_VERSION=1"
call :ReadCloud SECRET_RESOURCE secrets versions describe "%SECRET_VERSION%" --secret="%SECRET_NAME%" --format="value(name)" || exit /b 1
call :ReadCloud SECRET_STATE secrets versions describe "%SECRET_VERSION%" --secret="%SECRET_NAME%" --format="value(state)" || exit /b 1
if "%DRY_RUN%"=="1" (
  set "MUCHI_API_TOKEN=%SECRET_NAME%:1"
  exit /b 0
)
if not "%SECRET_STATE%"=="ENABLED" exit /b 1
for %%V in ("%SECRET_RESOURCE:/=\%") do set "SECRET_VERSION=%%~nxV"
set "MUCHI_API_TOKEN=%SECRET_NAME%:%SECRET_VERSION%"
exit /b 0

:UploadToken
call :ReadCloud EXISTING secrets list --filter="name:%SECRET_NAME%" --format="value(name.basename())" || exit /b 1
if not "%EXISTING%"=="%SECRET_NAME%" call :Cloud secrets create "%SECRET_NAME%" --replication-policy=automatic || exit /b 1
call :ReadCloud SECRET_RESOURCE secrets versions add "%SECRET_NAME%" --data-file="%TOKEN_FILE%" --format="value(name)" || exit /b 1
for %%V in ("%SECRET_RESOURCE:/=\%") do set "SECRET_VERSION=%%~nxV"
exit /b 0

:Accounts
set "STEP=Cuentas y Permisos"
echo 🐱 %STEP%
for %%A in (%API_ACCOUNT% %WORKER_ACCOUNT% %TASK_ACCOUNT% %BUILD_ACCOUNT%) do (
  call :EnsureAccount %%A || exit /b 1
 )
call :GrantProject "%API_EMAIL%" roles/datastore.user || exit /b 1
call :GrantProject "%API_EMAIL%" roles/cloudtasks.enqueuer || exit /b 1
call :GrantProject "%WORKER_EMAIL%" roles/datastore.user || exit /b 1
for %%R in (roles/logging.logWriter roles/artifactregistry.writer roles/storage.objectViewer) do (
  call :GrantProject "%BUILD_EMAIL%" %%R || exit /b 1
 )
for %%A in (%API_EMAIL% %WORKER_EMAIL%) do (
  call :Cloud secrets add-iam-policy-binding "%SECRET_NAME%" --member="serviceAccount:%%A" --role=roles/secretmanager.secretAccessor --condition=None --format=none || exit /b 1
 )
call :Cloud iam service-accounts add-iam-policy-binding "%TASK_EMAIL%" --member="serviceAccount:%API_EMAIL%" --role=roles/iam.serviceAccountUser --condition=None --format=none || exit /b 1
call :ReadCloud TASK_AGENT beta services identity create --service=cloudtasks.googleapis.com --format="value(email)" || exit /b 1
if "%DRY_RUN%"=="1" set "TASK_AGENT=service-PROJECT_NUMBER@gcp-sa-cloudtasks.iam.gserviceaccount.com"
if not defined TASK_AGENT exit /b 1
call :GrantProject "%TASK_AGENT%" roles/cloudtasks.serviceAgent || exit /b 1
exit /b 0

:EnsureAccount
call :ReadCloud EXISTING iam service-accounts list --filter="email=%~1@%PROJECT%.iam.gserviceaccount.com" --format="value(email)" || exit /b 1
if not defined EXISTING call :Cloud iam service-accounts create "%~1" || exit /b 1
exit /b 0

:GrantProject
call :Cloud projects add-iam-policy-binding "%PROJECT%" --member="serviceAccount:%~1" --role="%~2" --condition=None --format=none
exit /b %ERRORLEVEL%

:Database
set "STEP=Firestore y Cola"
echo 🐱 %STEP%
call :ReadCloud EXISTING firestore databases list --filter="name:(default)" --format="value(name)" || exit /b 1
if not defined EXISTING call :Cloud firestore databases create "--database=(default)" --location="%FIRESTORE_LOCATION%" --type=firestore-native || exit /b 1
for %%G in (searches items item_offers idempotency offer_cache) do (
  call :Cloud firestore fields ttls update expires_at --collection-group=%%G "--database=(default)" --enable-ttl --async || exit /b 1
 )
call :ReadCloud EXISTING tasks queues list --location="%REGION%" --filter="name:%TASK_QUEUE%" --format="value(name.basename())" || exit /b 1
set "QUEUE_ACTION=create"
if "%EXISTING%"=="%TASK_QUEUE%" set "QUEUE_ACTION=update"
call :Cloud tasks queues %QUEUE_ACTION% "%TASK_QUEUE%" --location="%REGION%" --max-dispatches-per-second="%DISPATCH_RATE%" --max-concurrent-dispatches="%CONCURRENT_TASKS%" || exit /b 1
exit /b 0

:Worker
set "STEP=Despliegue del Worker"
echo 🐱 %STEP%
call :Cloud functions deploy "%WORKER_NAME%" --gen2 --trigger-http --no-allow-unauthenticated ^
  --runtime="%RUNTIME%" --region="%REGION%" --source=. --entry-point=ProcessSearch --ignore-file=.gcloudignore ^
  --service-account="%WORKER_EMAIL%" --build-service-account="projects/%PROJECT%/serviceAccounts/%BUILD_EMAIL%" ^
  --memory="%MEMORY%" --timeout="%TIMEOUT%" --min-instances=0 --max-instances="%MAX_INSTANCES%" --concurrency=1 ^
  --set-env-vars="GOOGLE_CLOUD_PROJECT=%PROJECT%,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml" ^
  --set-secrets="MUCHI_API_TOKEN=%MUCHI_API_TOKEN%" --format=none || exit /b 1
call :ReadCloud WORKER_URL functions describe "%WORKER_NAME%" --gen2 --region="%REGION%" --format="value(serviceConfig.uri)" || exit /b 1
call :ReadCloud WORKER_SERVICE functions describe "%WORKER_NAME%" --gen2 --region="%REGION%" --format="value(serviceConfig.service)" || exit /b 1
if "%DRY_RUN%"=="1" (
  set "WORKER_URL=https://%WORKER_NAME%-PREVIEW.run.app"
  set "WORKER_SERVICE=%WORKER_NAME%"
)
if not defined WORKER_URL exit /b 1
if not defined WORKER_SERVICE exit /b 1
for %%V in ("%WORKER_SERVICE:/=\%") do set "WORKER_SERVICE=%%~nxV"
call :Cloud run services add-iam-policy-binding "%WORKER_SERVICE%" --region="%REGION%" --member="serviceAccount:%TASK_EMAIL%" --role=roles/run.invoker --condition=None --format=none || exit /b 1
echo ✅ Worker Privado Preparado
exit /b 0

:API
set "STEP=Despliegue de la API"
echo 🐱 %STEP%
call :Cloud functions deploy "%API_NAME%" --gen2 --trigger-http --allow-unauthenticated ^
  --runtime="%RUNTIME%" --region="%REGION%" --source=. --entry-point=ServeAPI --ignore-file=.gcloudignore ^
  --service-account="%API_EMAIL%" --build-service-account="projects/%PROJECT%/serviceAccounts/%BUILD_EMAIL%" ^
  --memory="%MEMORY%" --timeout="%TIMEOUT%" --min-instances=0 --max-instances="%MAX_INSTANCES%" --concurrency=1 ^
  --set-env-vars="GOOGLE_CLOUD_PROJECT=%PROJECT%,MUCHI_STORES_CONFIG=serverless_function_source_code/config/stores.yaml,MUCHI_TASK_REGION=%REGION%,MUCHI_TASK_QUEUE=%TASK_QUEUE%,MUCHI_TASK_URL=%WORKER_URL%,MUCHI_TASK_SERVICE_ACCOUNT=%TASK_EMAIL%" ^
  --set-secrets="MUCHI_API_TOKEN=%MUCHI_API_TOKEN%" --format=none || exit /b 1
call :ReadCloud API_URL functions describe "%API_NAME%" --gen2 --region="%REGION%" --format="value(serviceConfig.uri)" || exit /b 1
if "%DRY_RUN%"=="0" if not defined API_URL exit /b 1
exit /b 0

:Failed
echo ❌ %STEP% Falló
del /q "%DEPLOY_READ%" >nul 2>nul
popd
exit /b 1
:Done
if exist "%DEPLOY_READ%" del /q "%DEPLOY_READ%" >nul 2>nul
popd
exit /b 0
