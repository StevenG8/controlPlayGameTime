@echo off
setlocal

set "TASK_NAME=GameControlAutostart"
schtasks /Delete /F /TN "%TASK_NAME%"
if errorlevel 1 (
  echo [ERROR] failed to delete scheduled task "%TASK_NAME%"
  exit /b 1
)

echo [OK] scheduled task removed: "%TASK_NAME%"
exit /b 0
