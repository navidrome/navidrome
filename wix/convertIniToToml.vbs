Const ForReading = 1    
Const ForWriting = 2

sSourceFilename = Wscript.Arguments(0)
sTargetFilename = Wscript.Arguments(1)

Set oFSO = CreateObject("Scripting.FileSystemObject")
Set oFile = oFSO.OpenTextFile(sSourceFilename, ForReading)
sFileContent = oFile.ReadAll
oFile.Close

sNewFileContent = Replace(sFileContent, "[MSI_PLACEHOLDER_SECTION]" & vbCrLf, "")
If Not ( oFSO.FileExists(sTargetFilename) ) Then
    Set oFile = oFSO.CreateTextFile(sTargetFilename)
    oFile.Write sNewFileContent
    oFile.Close
End If
