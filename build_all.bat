@echo off
echo ==========================================
echo       Go PC Manager Build Script
echo ==========================================

echo.
echo [1/4] Building Agent...
cd agent
go build -o agent.exe main.go session_windows.go update.go
if %ERRORLEVEL% NEQ 0 (
    echo Agent build failed!
    pause
    exit /b %ERRORLEVEL%
)
echo Agent build success.

echo.
echo [2/4] Copying Agent to Setup folder...
copy /Y agent.exe cmd\setup\agent.exe
if %ERRORLEVEL% NEQ 0 (
    echo Failed to copy agent.exe to setup!
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo [2-1/4] Copying Agent to Server Updates folder...
if not exist ..\server\updates mkdir ..\server\updates
copy /Y agent.exe ..\server\updates\agent.exe
if %ERRORLEVEL% NEQ 0 (
    echo Failed to copy agent.exe to updates!
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo [3/4] Building Setup...
cd cmd\setup
go build -o ../../../setup.exe main.go
if %ERRORLEVEL% NEQ 0 (
    echo Setup build failed!
    pause
    exit /b %ERRORLEVEL%
)
echo Setup build success.

echo.
echo [4/4] Building Server...
cd ../../../server
go build -o gopc-server.exe main.go
if %ERRORLEVEL% NEQ 0 (
    echo Server build failed!
    pause
    exit /b %ERRORLEVEL%
)
echo Server build success.

cd ..
echo.
echo ==========================================
echo       All Builds Completed Successfully!
echo ==========================================
echo.
echo Output files:
echo - agent/agent.exe
echo - setup.exe
echo - server/gopc-server.exe
echo.
pause
