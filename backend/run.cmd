@echo off
rem Runs Muchi with Docker Compose. Pass serve or work; serve is the Default.
setlocal
chcp 65001 >nul
cd /d "%~dp0" || exit /b 1

set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=serve"

call :VerifyCommand || exit /b 1
call :SelectService
call :StartService
exit /b %errorlevel%

:VerifyCommand
if "%COMMAND%"=="serve" exit /b 0
if "%COMMAND%"=="work" exit /b 0
echo ❌ Unknown Command: %COMMAND%
echo Usage: run.cmd [serve^|work]
exit /b 1

:SelectService
if "%COMMAND%"=="serve" set "SERVICE=api"
if "%COMMAND%"=="work" set "SERVICE=worker"
exit /b 0

:StartService
echo ✅ Starting Muchi %COMMAND%
docker compose up --build -d "%SERVICE%"
if errorlevel 1 exit /b %errorlevel%
docker compose logs --follow --tail 50 "%SERVICE%" firestore
exit /b %errorlevel%
