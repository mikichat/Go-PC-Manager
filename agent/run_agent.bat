@echo off
echo Building Agent...
go build -o agent.exe .
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b %errorlevel%
)

echo Starting Agent...
agent.exe
pause
