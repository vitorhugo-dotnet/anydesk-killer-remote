@echo off
setlocal EnableExtensions

set "BASE_DIR=%~dp0"
set "AGENT_EXE=%BASE_DIR%anydesk-killer-agent-windows-amd64.exe"
set "CONFIG_FILE=%BASE_DIR%config.json"
set "HIDDEN_LAUNCHER=%BASE_DIR%run-agent-hidden.vbs"
set "STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "STARTUP_SHORTCUT=%STARTUP_DIR%\AnyDesk Killer Agent.lnk"

if not exist "%AGENT_EXE%" (
    echo ERROR: Agent executable not found:
    echo %AGENT_EXE%
    exit /b 1
)

if not exist "%CONFIG_FILE%" (
    echo ERROR: config.json not found:
    echo %CONFIG_FILE%
    exit /b 1
)

if not exist "%HIDDEN_LAUNCHER%" (
    echo ERROR: Hidden launcher not found:
    echo %HIDDEN_LAUNCHER%
    exit /b 1
)

if not exist "%STARTUP_DIR%" mkdir "%STARTUP_DIR%"

powershell.exe -NoLogo -NoProfile -NonInteractive -Command "$shell = New-Object -ComObject WScript.Shell; $shortcut = $shell.CreateShortcut($env:STARTUP_SHORTCUT); $shortcut.TargetPath = Join-Path $env:SystemRoot 'System32\wscript.exe'; $quote = [char]34; $shortcut.Arguments = $quote + $env:HIDDEN_LAUNCHER + $quote; $shortcut.WorkingDirectory = $env:BASE_DIR; $shortcut.IconLocation = $env:AGENT_EXE + ',0'; $shortcut.Description = 'Starts the AnyDesk Killer agent without a console window'; $shortcut.Save()"

if errorlevel 1 (
    echo ERROR: Failed to create the startup shortcut.
    exit /b 1
)

echo Startup enabled for the current Windows user.
echo The agent will start silently at the next logon.
echo Shortcut: %STARTUP_SHORTCUT%
exit /b 0
