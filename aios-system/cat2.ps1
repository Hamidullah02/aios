Get-Content 'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\validate.go' | Select-String -Pattern 'func OptionalString' -Context 0,5 | ForEach-Object { $_.ToString() }
