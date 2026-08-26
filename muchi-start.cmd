@echo off
rem
rem Muchi, de cero a corriendo en Windows. Tres tiempos: Prepara el entorno,
rem Instala lo que falte, Levanta el server. Nada mas.
rem
rem Es idempotente: la primera vez crea el .venv y baja las dependencias, las
rem siguientes arranca directo. Solo reinstala si requirements.txt cambio.
rem
rem   muchi-start.cmd                        -^> http://localhost:8501
rem   muchi-start.cmd --server.port 9123     -^> otro puerto
rem   muchi-start.cmd --server.address 127.0.0.1 -^> solo tu maquina
rem
rem Todo lo que le pases viaja tal cual a "streamlit run app.py".
rem
setlocal

set "ROOT=%~dp0"
set "ROOT=%ROOT:~0,-1%"
set "VENV=%ROOT%\.venv"
set "VENV_PYTHON=%VENV%\Scripts\python.exe"
set "DEPS_STAMP=%VENV%\.muchi-deps"

call :prepare_venv
if errorlevel 1 exit /b 1

call :install_requirements
if errorlevel 1 exit /b 1

call :run_streamlit %*
exit /b %ERRORLEVEL%


rem ------------------------------------------------- buscar un python usable
rem El launcher `py` es lo normal en Windows, pero no siempre esta: probamos
rem por orden y verificamos la version preguntandosela al propio interprete,
rem no parseando su --version.
:find_python
set "PY_CMD="
if defined MUCHI_PYTHON call :try_python "%MUCHI_PYTHON%"
if not defined PY_CMD call :try_python "py -3"
if not defined PY_CMD call :try_python "python"
if not defined PY_CMD call :try_python "python3"
goto :eof

:try_python
%~1 -c "import sys; raise SystemExit(0 if sys.version_info >= (3, 12) else 1)" >nul 2>&1
if errorlevel 1 goto :eof
set "PY_CMD=%~1"
goto :eof


rem ----------------------------------------------------------- el entorno solo
rem Si el .venv ya esta, no lo tocamos. Si no, lo creamos con el python que
rem encontramos.
:prepare_venv
if exist "%VENV_PYTHON%" goto :eof

call :find_python
if not defined PY_CMD (
    echo ~nya!~ No Encuentro Python 3.12 o Mas Nuevo.
    echo        Instalalo con: winget install --id Python.Python.3.12 -e
    echo        O Apuntame al tuyo con: set MUCHI_PYTHON=C:\ruta\a\python.exe
    exit /b 1
)

echo ~nya~ Creando el Entorno en .venv
%PY_CMD% -m venv "%VENV%"
if errorlevel 1 (
    echo ~nya!~ No Pude Crear el Entorno en "%VENV%"
    exit /b 1
)

"%VENV_PYTHON%" -m pip install --quiet --upgrade pip
goto :eof


rem ------------------------------------------------ las dependencias, una vez
rem La copia de requirements.txt queda guardada dentro del .venv. Mientras el
rem archivo no cambie, arrancar cuesta cero: no hay pip que resuelva nada.
:install_requirements
if exist "%DEPS_STAMP%" (
    fc /b "%DEPS_STAMP%" "%ROOT%\requirements.txt" >nul 2>&1
    if not errorlevel 1 goto :eof
)

echo ~nya~ Instalando Dependencias (esto pasa una sola vez)
"%VENV_PYTHON%" -m pip install --quiet --requirement "%ROOT%\requirements.txt"
if errorlevel 1 (
    echo ~nya!~ Fallo la Instalacion de Dependencias
    exit /b 1
)

copy /y "%ROOT%\requirements.txt" "%DEPS_STAMP%" >nul
goto :eof


rem ------------------------------------------------------------ y a levantarlo
:run_streamlit
echo ~nya~ Levantando Muchi -- Ctrl+C para Cortar
echo       El Navegador no se Abre solo: Copia la URL que Sale abajo
echo.
cd /d "%ROOT%"
"%VENV_PYTHON%" -m streamlit run app.py %*
goto :eof
