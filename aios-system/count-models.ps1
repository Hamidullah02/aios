Get-ChildItem -Recurse 'E:\sadi\projects\aiagent\aios-system\internal\models' -File -Filter '*.go' | ForEach-Object {
    $lines = (Get-Content $_.FullName | Measure-Object -Line).Lines
    Write-Host ("{0,5}  {1}" -f $lines, $_.FullName.Substring('E:\sadi\projects\aiagent\aios-system\'.Length))
}
