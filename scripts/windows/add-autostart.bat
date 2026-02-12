@echo off
setlocal

set "TASK_NAME=GameControlAutostart"
set "DIST_DIR=%~dp0"
for %%I in ("%DIST_DIR%") do set "DIST_DIR=%%~fI"

set "START_SCRIPT=%DIST_DIR%\start-background.bat"

if not exist "%START_SCRIPT%" (
  echo [ERROR] file not found: "%START_SCRIPT%"
  exit /b 1
)

set "TASK_CMD=cmd.exe /c \"\"%START_SCRIPT%\"\""
schtasks /Create /F /SC ONLOGON /TN "%TASK_NAME%" /TR "%TASK_CMD%"
if errorlevel 1 (
  echo [ERROR] failed to create scheduled task "%TASK_NAME%"
  exit /b 1
)

echo [OK] scheduled task created: "%TASK_NAME%"
exit /b 0
