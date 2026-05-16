# Axen installer script for Windows PowerShell

$ErrorActionPreference = 'Stop'

# Define repository info
$GitHubRepo = "harishphk/axen"
$BinaryName = "axen.exe"

# Detect Architecture
$Arch = "amd64"
if ([System.Environment]::Is64BitOperatingSystem -eq $false) {
    Write-Error "Unsupported architecture: 32-bit Windows is not supported."
}
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

Write-Host "Detecting latest release..."
# Get latest release from GitHub API
$ReleaseUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"
try {
    # Configure TLS 1.2
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $Response = Invoke-RestMethod -Uri $ReleaseUrl -UseBasicParsing
    $LatestTag = $Response.tag_name
} catch {
    # Fallback tag detection if rate-limited
    Write-Warning "GitHub API rate limit hit, detecting release via redirection..."
    $Headers = @{}
    $Request = [System.Net.WebRequest]::Create("https://github.com/$GitHubRepo/releases/latest")
    $Request.AllowAutoRedirect = $false
    try {
        $Response = $Request.GetResponse()
        $RedirectUrl = $Response.Headers["Location"]
        $LatestTag = $RedirectUrl.Split('/')[-1]
    } catch {
        Write-Error "Could not retrieve latest release tag."
    }
}

if (-not $LatestTag) {
    Write-Error "Could not parse latest release tag."
}

Write-Host "Installing Axen $LatestTag (windows/$Arch)..."

$VersionNum = $LatestTag.TrimStart('v')
$FileName = "axen_${VersionNum}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$GitHubRepo/releases/download/$LatestTag/$FileName"

# Create a temporary directory for extraction
$TempDir = Join-Path $env:TEMP "axen-installer"
if (Test-Path $TempDir) {
    Remove-Item $TempDir -Recurse -Force
}
New-Item -ItemType Directory -Path $TempDir | Out-Null

$ZipPath = Join-Path $TempDir $FileName

Write-Host "Downloading from $DownloadUrl..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing

Write-Host "Extracting..."
Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

# Determine target directory (install to $HOME\.local\bin by default)
$InstallDir = Join-Path $HOME ".local\bin"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

$SourceExe = Join-Path $TempDir $BinaryName
$TargetExe = Join-Path $InstallDir $BinaryName

Write-Host "Installing to $TargetExe..."
Copy-Item -Path $SourceExe -Destination $TargetExe -Force

# Clean up
Remove-Item $TempDir -Recurse -Force

Write-Host "Axen $LatestTag has been installed successfully to $TargetExe!"

# Check if Target Dir is in PATH
$PathValue = [System.Environment]::GetEnvironmentVariable("PATH", "User")
$Paths = $PathValue -split ";"
$IsInstalledInPath = $false
foreach ($P in $Paths) {
    if ($P.TrimEnd('\') -eq $InstallDir.TrimEnd('\')) {
        $IsInstalledInPath = $true
        break
    }
}

if (-not $IsInstalledInPath) {
    Write-Warning "The installation directory '$InstallDir' is not in your User PATH."
    Write-Host "Adding '$InstallDir' to User PATH environment variable..."
    $NewPath = $PathValue + ";" + $InstallDir
    [System.Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "PATH updated. Please restart your terminal/PowerShell window to use 'axen'."
}
