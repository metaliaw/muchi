@echo off
rem Consulta el Estado de la Publicidad en Windows. La Logica vive en el .sh y el bash lo busca run-bash.cmd.
call "%~dp0run-bash.cmd" check-ads.sh %*
exit /b %errorlevel%
