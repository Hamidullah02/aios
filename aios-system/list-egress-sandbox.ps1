Get-ChildItem -Recurse 'E:\sadi\projects\aiagent\aios-system\internal\egress','E:\sadi\projects\aiagent\aios-system\internal\sandbox' -File -Filter '*.go' | ForEach-Object {
    $_.FullName.Substring('E:\sadi\projects\aiagent\aios-system\'.Length)
}
