Get-ChildItem -Recurse 'E:\sadi\projects\aiagent\aios-system\internal\runtime' -File -Filter '*.go' | ForEach-Object {
    $lines = (Get-Content $_.FullName | Measure-Object -Line).Lines
    Write-Host ("{0,5}  {1}" -f $lines, $_.FullName.Substring('E:\sadi\projects\aiagent\aios-system\'.Length))
}
Write-Host "---"
Get-ChildItem -Path 'E:\sadi\projects\aiagent\aios-system\tests' -File -Filter 'runtime*.go' | ForEach-Object {
    $lines = (Get-Content $_.FullName | Measure-Object -Line).Lines
    Write-Host ("{0,5}  {1}" -f $lines, $_.FullName.Substring('E:\sadi\projects\aiagent\aios-system\'.Length))
}
