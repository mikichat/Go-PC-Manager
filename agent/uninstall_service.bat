@echo off
echo Stopping Service...
agent.exe -service stop

echo Uninstalling Service...
agent.exe -service uninstall
if %ERRORLEVEL% NEQ 0 (
    echo Service uninstallation failed. Run as Administrator.
    pause
    exit /b %ERRORLEVEL%
)

echo Agent Service Uninstalled Successfully!
pause
