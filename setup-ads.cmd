@echo off
rem Prepara la Publicidad en Windows. La Logica vive en el .sh y el bash lo busca run-bash.cmd.
call "%~dp0run-bash.cmd" setup-ads.sh %*
exit /b %errorlevel%
