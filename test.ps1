Write-Host "Testing server endpoints..." -ForegroundColor Green

Write-Host "`n1. Testing health endpoint:" -ForegroundColor Yellow
$health = Invoke-RestMethod -Uri "http://localhost:8080/health" -UseBasicParsing
Write-Host "Health: $($health | ConvertTo-Json)"

Write-Host "`n2. Testing software version:" -ForegroundColor Yellow
$version = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/softwareVer" -UseBasicParsing
Write-Host "Version: $($version | ConvertTo-Json)"

Write-Host "`n3. Testing TIR start:" -ForegroundColor Yellow
$start = Invoke-RestMethod -Uri "http://localhost:8080/api/v2/startTir" -Method Post -UseBasicParsing
Write-Host "Start TIR: $($start | ConvertTo-Json)"

Write-Host "`n4. Testing TIR start again (should fail):" -ForegroundColor Yellow
$start2 = Invoke-RestMethod -Uri "http://localhost:8080/api/v2/startTir" -Method Post -UseBasicParsing
Write-Host "Start TIR again: $($start2 | ConvertTo-Json)"

Write-Host "`n5. Testing TIR stop:" -ForegroundColor Yellow
$stop = Invoke-RestMethod -Uri "http://localhost:8080/api/v2/stopTir" -Method Post -UseBasicParsing
Write-Host "Stop TIR: $($stop | ConvertTo-Json)"

Write-Host "`n6. Testing TIR restart:" -ForegroundColor Yellow
$restart = Invoke-RestMethod -Uri "http://localhost:8080/api/v2/restartTir" -Method Post -UseBasicParsing
Write-Host "Restart TIR: $($restart | ConvertTo-Json)"