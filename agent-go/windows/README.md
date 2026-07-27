# Silent startup on Windows

Keep these files in the same directory:

- `anydesk-killer-agent-windows-amd64.exe`
- `config.json`
- `run-agent-hidden.vbs`
- `install-startup.bat`
- `uninstall-startup.bat`

Run `install-startup.bat` once. It creates an `AnyDesk Killer Agent.lnk` shortcut in the current user's Windows Startup folder. At the next logon, Windows runs `wscript.exe`, which launches the Go agent with window style `0`, so no terminal window is displayed.

The installer does not require administrator privileges and does not start a second agent immediately. Stop any manually started agent before logging out or restarting Windows.

To remove automatic startup, run `uninstall-startup.bat`.

If the directory is moved after installation, run `install-startup.bat` again so the shortcut points to the new location.

The agent continues writing to the `logFile` configured in `config.json`.
