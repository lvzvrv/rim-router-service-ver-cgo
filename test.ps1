Write-Host "Testing server endpoints..." -ForegroundColor Green

function Test-Endpoint {
    param($Name, $Uri, $Method = "GET", $Body = $null)
    
    Write-Host "`n$Name" -ForegroundColor Yellow
    Write-Host "Endpoint: $Method $Uri" -ForegroundColor Gray
    
    try {
        if ($Method -eq "POST") {
            $result = Invoke-RestMethod -Uri $Uri -Method $Method -UseBasicParsing
        } else {
            $result = Invoke-RestMethod -Uri $Uri -Method $Method -UseBasicParsing
        }
        
        Write-Host "Status: SUCCESS" -ForegroundColor Green
        Write-Host "Response: $($result | ConvertTo-Json -Compress)" -ForegroundColor White
        return $result
    }
    catch {
        Write-Host "Status: ERROR" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

Test-Endpoint -Name "1. Health check" -Uri "http://localhost:8080/health"
Test-Endpoint -Name "2. Software version" -Uri "http://localhost:8080/api/v1/softwareVer"
Test-Endpoint -Name "3. Start TIR" -Uri "http://localhost:8080/api/v2/startTir" -Method "POST"
Test-Endpoint -Name "4. Start TIR again (should fail)" -Uri "http://localhost:8080/api/v2/startTir" -Method "POST"
Test-Endpoint -Name "5. Stop TIR" -Uri "http://localhost:8080/api/v2/stopTir" -Method "POST"
Test-Endpoint -Name "6. Restart TIR" -Uri "http://localhost:8080/api/v2/restartTir" -Method "POST"

Write-Host "`nTest completed!" -ForegroundColor Green