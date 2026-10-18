$content = Get-Content 'E:\sadi\projects\aiagent\aios\docs\global\PRD.md'
for ($i = 0; $i -lt $content.Count; $i++) {
    if ($content[$i] -match '^### 11\.5 |^### 11\.9 |^### 12\.5\.5 |^### 12\.5\.2') {
        Write-Host "Line $($i+1): $($content[$i])"
    }
}
