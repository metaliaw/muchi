@echo off
where bash >nul 2>nul
if errorlevel 1 (
  echo Instala Git Bash para Ejecutar la Rotacion.
  exit /b 1
)
bash "%~dp0rotate-secret.sh" %*
exit /b %errorlevel%
