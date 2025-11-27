@echo off
echo Building Agent...
go build -o agent.exe main.go session_windows.go update.go
if %ERRORLEVEL% NEQ 0 (
    echo Build failed.
    pause
    exit /b %ERRORLEVEL%
)

echo Installing Service...
agent.exe -service install
if %ERRORLEVEL% NEQ 0 (
    echo Service installation failed. Run as Administrator.
    pause
    exit /b %ERRORLEVEL%
)

echo Starting Service...
agent.exe -service start
if %ERRORLEVEL% NEQ 0 (
    echo Failed to start service.
    pause
    exit /b %ERRORLEVEL%
)

echo Agent Service Installed and Started Successfully!
pause
