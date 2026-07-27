@echo off
setlocal EnableExtensions

set "STARTUP_SHORTCUT=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\AnyDesk Killer Agent.lnk"

if exist "%STARTUP_SHORTCUT%" (
    del /q "%STARTUP_SHORTCUT%"
    if errorlevel 1 (
        echo ERROR: Failed to remove the startup shortcut.
        exit /b 1
    )
    echo Startup disabled for the current Windows user.
) else (
    echo Startup shortcut was not installed.
)

exit /b 0
