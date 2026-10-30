@echo off
rem Builds the Muchi Binary once the Format, Vet and Test Gates pass.
setlocal
chcp 65001 >nul

set "BINARY=muchi-api.exe"

call :VerifyFormat || exit /b 1
call :VerifyVet || exit /b 1
call :VerifyTests || exit /b 1
call :BuildBinary || exit /b 1
exit /b 0

:VerifyFormat
for /f "delims=" %%f in ('gofmt -l cmd internal') do (
	echo %%f
	echo ❌ Format Check Failed
	exit /b 1
)
echo ✅ Format Check Passed
exit /b 0

:VerifyVet
go vet ./...
if errorlevel 1 (
	echo ❌ Vet Failed
	exit /b 1
)
echo ✅ Vet Passed
exit /b 0

:VerifyTests
go test ./...
if errorlevel 1 (
	echo ❌ Tests Failed
	exit /b 1
)
echo ✅ Tests Passed
exit /b 0

:BuildBinary
set CGO_ENABLED=0
go build -trimpath -ldflags="-s -w" -o "%BINARY%" ./cmd/muchi-api
if errorlevel 1 (
	echo ❌ Build Failed
	exit /b 1
)
echo ✅ Binary Built at .\%BINARY%
exit /b 0
