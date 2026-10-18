Get-Content 'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\agentspec.go',
                'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\decisiontrace.go',
                'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\envelope.go',
                'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\modelcapability.go',
                'E:\sadi\projects\aiagent\aios-system\internal\core\schemas\taskplan.go' |
  Select-String -Pattern 'json:"[a-z_]+"' |
  ForEach-Object { ($_.ToString() -split 'json:')[1].Trim() } |
  Sort-Object -Unique
