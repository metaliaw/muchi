@echo off
rem Lee el Token de Muchi en Windows. La Logica vive en el .sh y el bash lo busca run-bash.cmd.
call "%~dp0run-bash.cmd" get-secret.sh %*
exit /b %errorlevel%
