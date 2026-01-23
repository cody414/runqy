# runqy Windows installer
# Usage: iwr https://raw.githubusercontent.com/publikey/runqy/main/install.ps1 -useb | iex
#
# Or download and run:
#   Invoke-WebRequest -Uri "https://raw.githubusercontent.com/publikey/runqy/main/install.ps1" -OutFile install.ps1
#   .\install.ps1
#
# Environment variables:
#   $env:VERSION     - Specific version to install (default: latest)
#   $env:INSTALL_DIR - Installation directory (default: %LOCALAPPDATA%\runqy)

$ErrorActionPreference = "Stop"

$Repo = "publikey/runqy"
$BinaryName = "runqy"
$DefaultInstallDir = "$env:LOCALAPPDATA\runqy"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { $DefaultInstallDir }

function Write-Info { param($Message) Write-Host "[INFO] $Message" -ForegroundColor Green }
function Write-Warn { param($Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }
function Write-Err { param($Message) Write-Host "[ERROR] $Message" -ForegroundColor Red; exit 1 }
function Write-Header { param($Message) Write-Host "==> $Message" -ForegroundColor Cyan }

function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
        return $release.tag_name
    } catch {
        Write-Err "Failed to get latest version: $_"
    }
}

function Get-Architecture {
    if ([Environment]::Is64BitOperatingSystem) {
        return "amd64"
    } else {
        return "386"
    }
}

function Install-Runqy {
    Write-Header "runqy installer for Windows"

    $Arch = Get-Architecture
    $Version = if ($env:VERSION) { $env:VERSION } else { Get-LatestVersion }
    $VersionNum = $Version.TrimStart('v')

    Write-Info "Installing $BinaryName $Version for windows/$Arch"

    # Construct download URL
    $ArchiveName = "${BinaryName}_${VersionNum}_windows_${Arch}.zip"
    $DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$ArchiveName"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Info "Created directory: $InstallDir"
    }

    # Download to temp
    $TempFile = Join-Path $env:TEMP $ArchiveName
    Write-Info "Downloading from $DownloadUrl"

    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempFile -UseBasicParsing
    } catch {
        Write-Err "Failed to download: $_"
    }

    # Extract
    Write-Info "Extracting..."
    $TempDir = Join-Path $env:TEMP "runqy-install"
    if (Test-Path $TempDir) { Remove-Item $TempDir -Recurse -Force }
    Expand-Archive -Path $TempFile -DestinationPath $TempDir -Force

    # Move binary to install dir
    $BinaryPath = Join-Path $TempDir "$BinaryName.exe"
    if (-not (Test-Path $BinaryPath)) {
        # Try without .exe
        $BinaryPath = Join-Path $TempDir $BinaryName
    }
    Move-Item -Path $BinaryPath -Destination "$InstallDir\$BinaryName.exe" -Force

    # Cleanup
    Remove-Item $TempFile -Force -ErrorAction SilentlyContinue
    Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue

    # Add to PATH if not already there
    $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($CurrentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
        Write-Warn "Added $InstallDir to PATH (restart terminal to use)"
    }

    Write-Host ""
    Write-Header "Installation complete!"
    Write-Info "Binary: $InstallDir\$BinaryName.exe"
    Write-Info "Version: $Version"
    Write-Host ""
    Write-Info "Next steps:"
    Write-Host "  1. Restart your terminal (to update PATH)"
    Write-Host ""
    Write-Host "  2. Start Redis:"
    Write-Host "     docker run -d --name redis -p 6379:6379 redis:alpine"
    Write-Host ""
    Write-Host "  3. Set environment variables and start the server:"
    Write-Host '     $env:REDIS_HOST = "localhost"'
    Write-Host '     $env:REDIS_PASSWORD = ""'
    Write-Host '     $env:RUNQY_API_KEY = "dev-api-key"'
    Write-Host "     $BinaryName serve --sqlite"
    Write-Host ""
    Write-Info "Run '$BinaryName --help' to see all available commands"
}

Install-Runqy
