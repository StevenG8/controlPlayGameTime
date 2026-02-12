@echo off
setlocal

set "DIST_DIR=%~dp0"
for %%I in ("%DIST_DIR%.") do set "DIST_DIR=%%~fI"

set "EXE_PATH=%DIST_DIR%\game-control.exe"
set "CONFIG_PATH=%DIST_DIR%\config.yaml"

if not exist "%EXE_PATH%" (
  echo [ERROR] file not found: "%EXE_PATH%"
  exit /b 1
)

if not exist "%CONFIG_PATH%" (
  echo [ERROR] file not found: "%CONFIG_PATH%"
  exit /b 1
)

set "PS_CMD=Start-Process -FilePath '%EXE_PATH%' -ArgumentList 'start','%CONFIG_PATH%' -WorkingDirectory '%DIST_DIR%' -WindowStyle Hidden"
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -Command "%PS_CMD%"
if errorlevel 1 (
  echo [ERROR] failed to start process with PowerShell
  exit /b 1
)

exit /b 0
