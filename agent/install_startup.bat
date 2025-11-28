@echo off
set "SCRIPT_DIR=%~dp0"
set "SHORTCUT_PATH=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\GoPCAgent.lnk"
set "TARGET_PATH=%SCRIPT_DIR%run_agent.bat"

echo Creating shortcut at %SHORTCUT_PATH%
echo Target: %TARGET_PATH%

powershell "$s=(New-Object -COM WScript.Shell).CreateShortcut('%SHORTCUT_PATH%');$s.TargetPath='%TARGET_PATH%';$s.WorkingDirectory='%SCRIPT_DIR%';$s.Save()"

if %errorlevel% equ 0 (
    echo Successfully added to Startup.
) else (
    echo Failed to add to Startup.
)
pause
