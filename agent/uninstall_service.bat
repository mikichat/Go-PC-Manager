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
