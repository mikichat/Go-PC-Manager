@echo off
set "SHORTCUT_PATH=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\GoPCAgent.lnk"

if exist "%SHORTCUT_PATH%" (
    del "%SHORTCUT_PATH%"
    echo Successfully removed from Startup.
) else (
    echo Startup shortcut not found.
)
pause
