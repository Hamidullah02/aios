$content = Get-Content 'E:\sadi\projects\aiagent\aios\docs\global\PRD.md'
$indices = @()
for ($i = 0; $i -lt $content.Count; $i++) {
    if ($content[$i] -match '^### 11\.[0-9]+') { $indices += $i }
}
for ($j = 0; $j -lt $indices.Length - 1; $j++) {
    Write-Host "--- Section starting at line $($indices[$j]+1): $($content[$indices[$j]]) ---"
    for ($k = $indices[$j] + 1; $k -lt $indices[$j+1]; $k++) {
        Write-Host $content[$k]
    }
}
Write-Host "--- Last section starting at line $($indices[-1]+1): $($content[$indices[-1]]) ---"
for ($k = $indices[-1] + 1; $k -lt [Math]::Min($indices[-1] + 30, $content.Count); $k++) {
    Write-Host $content[$k]
}
