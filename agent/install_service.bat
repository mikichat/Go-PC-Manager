@echo off
setlocal

:: Check for Administrator privileges
>nul 2>&1 "%SYSTEMROOT%\system32\cacls.exe" "%SYSTEMROOT%\system32\config\system"
if '%errorlevel%' NEQ '0' (
    echo Requesting administrative privileges...
    goto UACPrompt
) else ( goto gotAdmin )

:UACPrompt
    echo Set UAC = CreateObject^("Shell.Application"^) > "%temp%\getadmin.vbs"
    echo UAC.ShellExecute "%~s0", "", "", "runas", 1 >> "%temp%\getadmin.vbs"
    "%temp%\getadmin.vbs"
    exit /B

:gotAdmin
    if exist "%temp%\getadmin.vbs" ( del "%temp%\getadmin.vbs" )
    pushd "%CD%"
    CD /D "%~dp0"

:: Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: Go is not installed or not in PATH.
    echo Please install Go from https://go.dev/dl/
    pause
    exit /b 1
)

echo Cleaning up previous build...
if exist agent.exe del agent.exe

echo Building Agent...
go build -o agent.exe main.go session_windows.go update.go
if %ERRORLEVEL% NEQ 0 (
    echo Build failed.
    pause
    exit /b %ERRORLEVEL%
)

if not exist agent.exe (
    echo Build failed: agent.exe was not created.
    pause
    exit /b 1
)

echo Installing Service...
agent.exe -service install
if %ERRORLEVEL% NEQ 0 (
    echo Service installation failed. Please Run as Administrator.
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
