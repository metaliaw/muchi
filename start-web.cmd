@echo off
rem Levanta Muchi en local en Windows. La Logica vive en el .sh y el bash lo busca run-bash.cmd.
call "%~dp0run-bash.cmd" start-web.sh %*
exit /b %errorlevel%
