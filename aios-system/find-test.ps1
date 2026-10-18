Get-Content 'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\extras2_test.go' | Select-String -Pattern 'TestBudget_' | ForEach-Object { $_.ToString() }
