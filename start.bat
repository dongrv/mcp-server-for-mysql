@echo off
setlocal

set "COMMAND=%~1"
if "%COMMAND%"=="" set "COMMAND=run"
set "CONFIG_PATH=%~2"
if "%CONFIG_PATH%"=="" set "CONFIG_PATH=config.yaml"
set "BINARY=build\mcp-database.exe"

if "%COMMAND%"=="build" goto build
if "%COMMAND%"=="test" goto test
if "%COMMAND%"=="run" goto run

echo Usage: start.bat [build^|test^|run] [config-path]
echo The configuration defaults to config.yaml and DSNs are supplied through its environment references.
exit /b 2

:build
if not exist build mkdir build
go build -o "%BINARY%" ./cmd
exit /b %errorlevel%

:test
go test ./...
exit /b %errorlevel%

:run
if not exist "%BINARY%" (
  if not exist build mkdir build
  go build -o "%BINARY%" ./cmd
  if errorlevel 1 exit /b 1
)
"%BINARY%" -config "%CONFIG_PATH%"
exit /b %errorlevel%
