Write-Host "=============================================="
Write-Host "       LPU Zero-Installation Setup            "
Write-Host "=============================================="

$Url = "https://github.com/lpu-software/rtm/releases/latest/download/lpu.exe"
$InstallDir = "$env:USERPROFILE\AppData\Local\lpu"
$Output = "$InstallDir\lpu.exe"

Write-Host "Downloading LPU for Windows..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-WebRequest -Uri $Url -OutFile $Output

if (Test-Path $Output) {
    Write-Host "Adding LPU to System PATH..."
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
    }

    Write-Host "Installation complete! You can now run:"
    Write-Host "  lpu start         (to share your screen in the background)"
    Write-Host "  lpu lele          (to share your screen in the foreground)"
    Write-Host "  lpu dede <code>   (to connect to another computer)"
    Write-Host "=============================================="
} else {
    Write-Host "Error: Failed to download LPU."
}
