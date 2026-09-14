@echo off
rem
rem Corre un Script de Muchi con el bash que haya, y en ese Orden.
rem
rem   run-bash.cmd deploy.sh [Argumentos]
rem
rem Lo llaman los otros .cmd, que solo pasan su Nombre. La Busqueda vive aca
rem una vez: cinco Copias de esta Logica se desincronizan a la primera Ruta
rem nueva de Git, y el que la arregla no se acuerda de las otras cuatro.
rem
rem Git Bash primero porque comparte el Sistema de Archivos y el PATH de
rem Windows: gcloud, npm y firebase estan donde el Script los espera. WSL es
rem el Respaldo, y es otro Sistema: %~dp0 no significa nada ahi, la Ruta hay
rem que traducirla, y las Herramientas tienen que estar instaladas adentro.
rem
setlocal enabledelayedexpansion

set "SCRIPT=%~1"
if "%SCRIPT%"=="" (
  echo run-bash.cmd espera el Nombre de un Script.
  exit /b 2
)
shift

rem shift no mueve %*, asi que los Argumentos se rearman de a uno. Se usa %1 y
rem no %~1 para conservar las Comillas de un Valor con Espacios.
set "ARGS="
:siguiente
if "%~1"=="" goto listo
set "ARGS=!ARGS! %1"
shift
goto siguiente
:listo

set "DIR=%~dp0"
set "DIR=%DIR:~0,-1%"

rem ------------------------------------------------------------- Git Bash
rem Las tres Rutas de siempre, y despues el bash que acompana al git del PATH:
rem quien tiene git tiene bash al lado, aunque lo haya instalado en otra parte.
set "GITBASH="
if exist "%ProgramFiles%\Git\bin\bash.exe" set "GITBASH=%ProgramFiles%\Git\bin\bash.exe"
if not defined GITBASH if exist "%ProgramFiles(x86)%\Git\bin\bash.exe" set "GITBASH=%ProgramFiles(x86)%\Git\bin\bash.exe"
if not defined GITBASH if exist "%LOCALAPPDATA%\Programs\Git\bin\bash.exe" set "GITBASH=%LOCALAPPDATA%\Programs\Git\bin\bash.exe"
if not defined GITBASH (
  for /f "delims=" %%G in ('where git 2^>nul') do (
    if not defined GITBASH if exist "%%~dpG..\bin\bash.exe" set "GITBASH=%%~dpG..\bin\bash.exe"
  )
)

if defined GITBASH (
  "%GITBASH%" "%DIR%\%SCRIPT%"%ARGS%
  exit /b %errorlevel%
)

rem ----------------------------------------------------------------- el PATH
rem Un bash del PATH puede ser el Lanzador de WSL que vive en System32. Se
rem descarta a proposito: entra por la Rama de abajo, que si traduce la Ruta.
for /f "delims=" %%B in ('where bash 2^>nul') do (
  if not defined PATHBASH (
    echo %%B | find /i "\System32\" >nul || set "PATHBASH=%%B"
  )
)
if defined PATHBASH (
  "%PATHBASH%" "%DIR%\%SCRIPT%"%ARGS%
  exit /b %errorlevel%
)

rem -------------------------------------------------------------------- WSL
rem wslpath traduce C:\... a /mnt/c/...: sin eso el Script no se encuentra.
rem -l carga el Perfil, porque gcloud y npm suelen vivir en un PATH que solo
rem el Perfil arma.
where wsl >nul 2>nul
if not errorlevel 1 (
  for /f "delims=" %%W in ('wsl -e wslpath -a "%DIR%" 2^>nul') do set "WSLDIR=%%W"
  if defined WSLDIR (
    echo Sin Git Bash: corriendo en WSL. Las Herramientas ^(gcloud, npm, firebase^)
    echo tienen que estar instaladas dentro de WSL, no en Windows.
    wsl -e bash -lc "cd '!WSLDIR!' && ./%SCRIPT%%ARGS%"
    exit /b !errorlevel!
  )
)

echo No Encuentro bash. Instala Git para Windows: https://git-scm.com/download/win
echo Tambien sirve WSL: wsl --install
exit /b 1
