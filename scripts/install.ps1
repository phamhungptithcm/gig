param(
    [string]$Version = "",
    [string]$Repo = "",
    [string]$InstallDir = "",
    [int]$WaitForPid = 0
)

$ErrorActionPreference = "Stop"

function Normalize-Version {
    param([string]$Value)

    if (-not $Value -or $Value -eq "latest") {
        return "latest"
    }

    if ($Value.StartsWith("v", [System.StringComparison]::OrdinalIgnoreCase)) {
        return "v" + $Value.Substring(1)
    }

    return "v$Value"
}

function Get-ReleaseMetadata {
    param(
        [string]$RepoName,
        [string]$RequestedVersion
    )

    $releaseUrl = if ($RequestedVersion -eq "latest") {
        "https://api.github.com/repos/$RepoName/releases/latest"
    } else {
        "https://api.github.com/repos/$RepoName/releases/tags/$RequestedVersion"
    }

    Invoke-RestMethod -Uri $releaseUrl
}

function Get-ReleaseAsset {
    param(
        [object]$Release,
        [string[]]$Names
    )

    foreach ($name in $Names) {
        $asset = $Release.assets | Where-Object { $_.name -eq $name } | Select-Object -First 1
        if ($asset) {
            return $asset
        }
    }

    return $null
}

function Get-ExpectedHash {
    param(
        [object]$Release,
        [object]$Asset,
        [string]$ResolvedVersion
    )

    if ($Asset.PSObject.Properties.Name -contains "digest" -and $Asset.digest -match "^sha256:(.+)$") {
        return $Matches[1].ToLowerInvariant()
    }

    $checksumAssetName = "gig_$($ResolvedVersion.TrimStart('v'))_checksums.txt"
    $checksumAsset = Get-ReleaseAsset -Release $Release -Names @($checksumAssetName)
    if (-not $checksumAsset) {
        return $null
    }

    $checksums = Invoke-RestMethod -Uri $checksumAsset.browser_download_url
    foreach ($line in ($checksums -split "`n")) {
        $trimmed = $line.Trim()
        if (-not $trimmed) {
            continue
        }

        $parts = $trimmed -split "\s+", 2
        if ($parts.Count -eq 2 -and $parts[1] -eq $Asset.name) {
            return $parts[0].ToLowerInvariant()
        }
    }

    return $null
}

if (-not $Version) {
    $Version = if ($env:GIG_VERSION) { $env:GIG_VERSION } else { "latest" }
}
$Version = Normalize-Version $Version

if (-not $Repo) {
    $Repo = if ($env:GIG_REPO) { $env:GIG_REPO } else { "phamhungptithcm/gig" }
}

if (-not $InstallDir) {
    $InstallDir = if ($env:GIG_INSTALL_DIR) { $env:GIG_INSTALL_DIR } else { "$HOME\bin" }
}

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
switch ($arch.ToString()) {
    "X64" { $stableAsset = "gig_windows_amd64.zip" }
    "Arm64" { $stableAsset = "gig_windows_arm64.zip" }
    default { throw "Unsupported Windows architecture: $arch" }
}

if ($WaitForPid -gt 0) {
    for ($attempt = 0; $attempt -lt 240; $attempt++) {
        if (-not (Get-Process -Id $WaitForPid -ErrorAction SilentlyContinue)) {
            break
        }
        Start-Sleep -Milliseconds 500
    }
}

$release = Get-ReleaseMetadata -RepoName $Repo -RequestedVersion $Version
$resolvedVersion = $release.tag_name
if (-not $resolvedVersion) {
    throw "Failed to resolve the requested gig release from GitHub."
}

$versionedAsset = "gig_$($resolvedVersion.TrimStart('v'))_$($stableAsset.Substring(4))"
$asset = Get-ReleaseAsset -Release $release -Names @($stableAsset, $versionedAsset)
if (-not $asset) {
    throw "No Windows asset matched $stableAsset or $versionedAsset in $resolvedVersion."
}

$expectedHash = Get-ExpectedHash -Release $release -Asset $asset -ResolvedVersion $resolvedVersion

$tmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("gig-install-" + [Guid]::NewGuid().ToString("N"))
$archivePath = Join-Path $tmpRoot $asset.name
$extractPath = Join-Path $tmpRoot "extract"

New-Item -ItemType Directory -Force -Path $tmpRoot | Out-Null
New-Item -ItemType Directory -Force -Path $extractPath | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

try {
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archivePath

    if ($expectedHash) {
        $actualHash = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
        if ($actualHash -ne $expectedHash) {
            throw "Checksum verification failed for $($asset.name). Expected $expectedHash but got $actualHash."
        }
    }

    Expand-Archive -Path $archivePath -DestinationPath $extractPath -Force

    $binarySource = Join-Path $extractPath "gig.exe"
    $binaryTarget = Join-Path $InstallDir "gig.exe"
    $action = if (Test-Path $binaryTarget) { "updated" } else { "installed" }

    Copy-Item -Path $binarySource -Destination $binaryTarget -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @()
    if ($userPath) {
        $pathEntries = $userPath.Split(';') | Where-Object { $_ -ne "" }
    }

    if ($pathEntries -notcontains $InstallDir) {
        $newUserPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Host "Added $InstallDir to the user PATH."
    }

    Write-Host "gig $action to $binaryTarget"
    Write-Host ""
    & $binaryTarget version
    Write-Host ""
    Write-Host "Open a new terminal if 'gig' is not available in the current session."
} finally {
    Remove-Item -Path $tmpRoot -Recurse -Force -ErrorAction SilentlyContinue
}
