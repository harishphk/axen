# Axen installer script for Windows PowerShell

$ErrorActionPreference = 'Stop'

# Define repository info
$GitHubRepo = "harishphk/axen"
$BinaryName = "axen.exe"

function Get-PlatformArch {
    if ([System.Environment]::Is64BitOperatingSystem -eq $false) {
        Write-Error "Unsupported architecture: 32-bit Windows is not supported."
    }
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
        return "arm64"
    }
    return "amd64"
}

function Get-LatestTag {
    Write-Host "Detecting latest release..."
    $ReleaseUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $Response = Invoke-RestMethod -Uri $ReleaseUrl -UseBasicParsing
        return $Response.tag_name
    } catch {
        Write-Warning "GitHub API rate limit hit, detecting release via redirection..."
        $Request = [System.Net.WebRequest]::Create("https://github.com/$GitHubRepo/releases/latest")
        $Request.AllowAutoRedirect = $false
        try {
            $Response = $Request.GetResponse()
            $RedirectUrl = $Response.Headers["Location"]
            return $RedirectUrl.Split('/')[-1]
        } catch {
            Write-Error "Could not retrieve latest release tag."
        }
    }
}

function Test-AlreadyUpToDate($latestTag) {
    $force = $env:AXEN_FORCE -eq "1"
    if (-not $force -and (Get-Command axen -ErrorAction SilentlyContinue)) {
        try {
            $currentOutput = & axen --version 2>$null
            $currentVersion = ($currentOutput -split "\s+")[-1]
            $versionNum = $latestTag.TrimStart('v')
            if ($currentVersion -and ($currentVersion -eq $latestTag -or $currentVersion -eq $versionNum)) {
                Write-Host "Axen is already up to date ($currentVersion)."
                Write-Host "To force reinstall, run: `$env:AXEN_FORCE = '1'; irm https://raw.githubusercontent.com/$GitHubRepo/main/install.ps1 | iex"
                return $true
            }
        } catch {}
    }
    return $false
}

function Update-UserPath($installDir) {
    $pathValue = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    $paths = $pathValue -split ";"
    $isInstalledInPath = $false
    foreach ($p in $paths) {
        if ($p.TrimEnd('\') -eq $installDir.TrimEnd('\')) {
            $isInstalledInPath = $true
            break
        }
    }

    if (-not $isInstalledInPath) {
        Write-Warning "The installation directory '$installDir' is not in your User PATH."
        Write-Host "Adding '$installDir' to User PATH environment variable..."
        $newPath = $pathValue + ";" + $installDir
        [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Host "PATH updated. Please restart your terminal/PowerShell window to use 'axen'."
    }
}

function Install-Axen {
    $arch = Get-PlatformArch
    $latestTag = Get-LatestTag
    if (-not $latestTag) {
        Write-Error "Could not parse latest release tag."
    }

    if (Test-AlreadyUpToDate $latestTag) {
        return
    }

    $versionNum = $latestTag.TrimStart('v')
    Write-Host "Installing Axen $latestTag (windows/$arch)..."

    $fileName = "axen_${versionNum}_windows_${arch}.zip"
    $downloadUrl = "https://github.com/$GitHubRepo/releases/download/$latestTag/$fileName"
    $checksumUrl = "https://github.com/$GitHubRepo/releases/download/$latestTag/checksums.txt"

    # Use a unique process temp directory
    $tempDir = Join-Path $env:TEMP ("axen-installer-" + [System.Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $tempDir | Out-Null

    try {
        $zipPath = Join-Path $tempDir $fileName
        $checksumPath = Join-Path $tempDir "checksums.txt"

        Write-Host "Downloading from $downloadUrl..."
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing

        Write-Host "Verifying checksum..."
        try {
            Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing
            $expectedHash = ""
            Get-Content $checksumPath | ForEach-Object {
                $parts = $_ -split "\s+"
                if ($parts.Length -ge 2 -and $parts[1].Trim() -eq $fileName) {
                    $expectedHash = $parts[0].Trim().ToLower()
                }
            }
            if ($expectedHash) {
                $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
                if ($actualHash -ne $expectedHash) {
                    Write-Error "Checksum verification failed! Expected: $expectedHash, got: $actualHash"
                }
            }
        } catch {
            Write-Warning "Could not verify checksum: $_"
        }

        Write-Host "Extracting..."
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

        $installDir = Join-Path $HOME ".local\bin"
        if (-not (Test-Path $installDir)) {
            New-Item -ItemType Directory -Path $installDir | Out-Null
        }

        $sourceExe = Join-Path $tempDir $BinaryName
        $targetExe = Join-Path $installDir $BinaryName

        Write-Host "Installing to $targetExe..."
        Copy-Item -Path $sourceExe -Destination $targetExe -Force

        Write-Host "Axen $latestTag has been installed successfully to $targetExe!"
        Update-UserPath $installDir
    } finally {
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Install-Axen
