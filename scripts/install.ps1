param(
    [string]$Version = "",
    [string]$Repo = "",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

if (-not $Version) {
    $Version = if ($env:GIG_VERSION) { $env:GIG_VERSION } else { "latest" }
}

if (-not $Repo) {
    $Repo = if ($env:GIG_REPO) { $env:GIG_REPO } else { "phamhungptithcm/gig" }
}

if (-not $InstallDir) {
    $InstallDir = if ($env:GIG_INSTALL_DIR) { $env:GIG_INSTALL_DIR } else { "$HOME\bin" }
}

if ($Version -ne "latest" -and -not $Version.StartsWith("v")) {
    $Version = "v$Version"
}

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
switch ($arch.ToString()) {
    "X64" { $asset = "gig_windows_amd64.zip" }
    "Arm64" { $asset = "gig_windows_arm64.zip" }
    default { throw "Unsupported Windows architecture: $arch" }
}

if ($Version -eq "latest") {
    $url = "https://github.com/$Repo/releases/latest/download/$asset"
} else {
    $url = "https://github.com/$Repo/releases/download/$Version/$asset"
}

$tmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("gig-install-" + [Guid]::NewGuid().ToString("N"))
$archivePath = Join-Path $tmpRoot $asset
$extractPath = Join-Path $tmpRoot "extract"

New-Item -ItemType Directory -Force -Path $tmpRoot | Out-Null
New-Item -ItemType Directory -Force -Path $extractPath | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

try {
    Invoke-WebRequest -Uri $url -OutFile $archivePath
    Expand-Archive -Path $archivePath -DestinationPath $extractPath -Force

    $binarySource = Join-Path $extractPath "gig.exe"
    $binaryTarget = Join-Path $InstallDir "gig.exe"

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

    Write-Host "gig installed to $binaryTarget"
    Write-Host ""
    & $binaryTarget version
    Write-Host ""
    Write-Host "Open a new terminal if 'gig' is not available in the current session."
} finally {
    Remove-Item -Path $tmpRoot -Recurse -Force -ErrorAction SilentlyContinue
}
