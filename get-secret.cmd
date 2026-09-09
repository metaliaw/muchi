@echo off
where bash >nul 2>nul
if errorlevel 1 (
  echo Instala Git Bash para Leer el Secreto.
  exit /b 1
)
bash "%~dp0get-secret.sh" %*
exit /b %errorlevel%
