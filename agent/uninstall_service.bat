@echo off
setlocal

if not exist agent.exe (
    echo agent.exe not found. Cannot uninstall service using agent self-uninstall.
    echo Trying sc delete...
    sc delete GoPCAgent
    if %ERRORLEVEL% NEQ 0 (
        echo Failed to delete service using sc.
    ) else (
        echo Service deleted using sc.
    )
    pause
    exit /b
)

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
