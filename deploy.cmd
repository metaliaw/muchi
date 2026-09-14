@echo off
rem Despliega Muchi en Windows. La Logica vive en el .sh y el bash lo busca run-bash.cmd.
call "%~dp0run-bash.cmd" deploy.sh %*
exit /b %errorlevel%
