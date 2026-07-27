Option Explicit

Dim fileSystem, shell, baseDirectory, agentPath, configPath, command

Set fileSystem = CreateObject("Scripting.FileSystemObject")
Set shell = CreateObject("WScript.Shell")

baseDirectory = fileSystem.GetParentFolderName(WScript.ScriptFullName)
agentPath = fileSystem.BuildPath(baseDirectory, "anydesk-killer-agent-windows-amd64.exe")
configPath = fileSystem.BuildPath(baseDirectory, "config.json")

If Not fileSystem.FileExists(agentPath) Then
    WScript.Quit 2
End If

If Not fileSystem.FileExists(configPath) Then
    WScript.Quit 3
End If

shell.CurrentDirectory = baseDirectory
command = Chr(34) & agentPath & Chr(34) & " --config " & Chr(34) & configPath & Chr(34)
shell.Run command, 0, False
