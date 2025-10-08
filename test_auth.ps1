Write-Host "Testing authentication endpoints..." -ForegroundColor Green

$baseUrl = "http://localhost:8080"

function Test-Endpoint {
    param($Name, $Uri, $Method = "GET", $Body = $null, $Token = $null)
    
    Write-Host "`n$Name" -ForegroundColor Yellow
    Write-Host "Endpoint: $Method $Uri" -ForegroundColor Gray
    
    $headers = @{}
    if ($Token) {
        $headers["Authorization"] = "Bearer $Token"
    }
    
    try {
        if ($Method -eq "POST") {
            $result = Invoke-RestMethod -Uri $Uri -Method $Method -Body ($Body | ConvertTo-Json) -ContentType "application/json" -Headers $headers -UseBasicParsing
        } else {
            $result = Invoke-RestMethod -Uri $Uri -Method $Method -Headers $headers -UseBasicParsing
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

# 1. Регистрация нового пользователя
Test-Endpoint -Name "1. Register user" -Uri "$baseUrl/api/v1/register" -Method "POST" -Body @{
    username = "testuser"
    password = "testpass123"
}

# 2. Вход пользователя
$loginResult = Test-Endpoint -Name "2. Login user" -Uri "$baseUrl/api/v1/login" -Method "POST" -Body @{
    username = "testuser"
    password = "testpass123"
}

$token = $loginResult.data.token

# 3. Доступ к защищенному endpoint с токеном
Test-Endpoint -Name "3. Access protected endpoint" -Uri "$baseUrl/api/v1/softwareVer" -Token $token

# 4. Попытка доступа к admin endpoint (должна fail для обычного пользователя)
Test-Endpoint -Name "4. Access admin endpoint (should fail)" -Uri "$baseUrl/api/v2/startTir" -Method "POST" -Token $token

Write-Host "`nAuth test completed!" -ForegroundColor Green