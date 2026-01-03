#!/usr/bin/env pwsh
# The MIT License (MIT)
# Copyright (c) 2026 Marcel Joachim Kloubert <https://marcel.coffee>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
# of the Software, and to permit persons to whom the Software is furnished to do
# so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
# FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

#Requires -Version 5.1

$ErrorActionPreference = "Stop"

# =============================================================================
# Configuration
# =============================================================================

$Script:AutoarkRepoUrl = if ($env:AUTARK_REPO_URL) { $env:AUTARK_REPO_URL } else { "https://github.com/mkloubert/autark.git" }
$Script:AutoarkPkgMgr = $env:AUTARK_PKG_MGR
$Script:AutoarkBin = $env:AUTARK_BIN
$Script:GoDownloadUrl = "https://go.dev/dl/?mode=json"
$Script:TempDir = $null
$Script:GoOs = $null
$Script:GoArch = $null
$Script:PkgMgr = $null
$Script:GoBin = $null

# =============================================================================
# Utility Functions
# =============================================================================

function Write-LogInfo {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-LogError {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-LogSuccess {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Invoke-Cleanup {
    if ($Script:TempDir -and (Test-Path $Script:TempDir)) {
        Write-LogInfo "Cleaning up temporary directory..."
        Remove-Item -Path $Script:TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Test-CommandExists {
    param([string]$Command)
    $null -ne (Get-Command $Command -ErrorAction SilentlyContinue)
}

# =============================================================================
# Phase 1: System Validation and Prerequisites
# =============================================================================

function Test-AdminPrivileges {
    Write-LogInfo "Checking for admin/root privileges..."

    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
        $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

        if (-not $isAdmin) {
            Write-LogError "This script must be run as Administrator."
            Write-LogError "Please right-click PowerShell and select 'Run as Administrator'."
            exit 1
        }
    }
    else {
        # Linux/macOS with PowerShell Core
        $uid = & id -u 2>$null
        if ($uid -ne "0") {
            Write-LogError "This script must be run as root (use sudo)."
            exit 1
        }
    }

    Write-LogInfo "Admin/root privileges confirmed."
}

function Get-OperatingSystem {
    Write-LogInfo "Detecting operating system..."

    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        $Script:GoOs = "windows"
    }
    elseif ($IsMacOS) {
        $Script:GoOs = "darwin"
    }
    elseif ($IsLinux) {
        $Script:GoOs = "linux"
    }
    else {
        # Fallback for Windows PowerShell 5.1
        if ([System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT) {
            $Script:GoOs = "windows"
        }
        else {
            Write-LogError "Unsupported operating system."
            exit 1
        }
    }

    Write-LogInfo "Detected OS: $Script:GoOs"
}

function Get-Architecture {
    Write-LogInfo "Detecting processor architecture..."

    $arch = $null

    if ($IsWindows -or $env:OS -eq "Windows_NT" -or [System.Environment]::OSVersion.Platform -eq [System.PlatformID]::Win32NT) {
        $arch = $env:PROCESSOR_ARCHITECTURE
        switch ($arch) {
            "AMD64" { $Script:GoArch = "amd64" }
            "ARM64" { $Script:GoArch = "arm64" }
            default {
                Write-LogError "Unsupported architecture: $arch. Only 64-bit systems are supported."
                exit 1
            }
        }
    }
    else {
        # Linux/macOS
        $uname = & uname -m 2>$null
        switch ($uname) {
            "x86_64" { $Script:GoArch = "amd64" }
            "amd64" { $Script:GoArch = "amd64" }
            "aarch64" { $Script:GoArch = "arm64" }
            "arm64" { $Script:GoArch = "arm64" }
            "ppc64le" { $Script:GoArch = "ppc64le" }
            "ppc64" { $Script:GoArch = "ppc64" }
            "s390x" { $Script:GoArch = "s390x" }
            "riscv64" { $Script:GoArch = "riscv64" }
            default {
                Write-LogError "Unsupported architecture: $uname. Only 64-bit systems are supported."
                exit 1
            }
        }
    }

    Write-LogInfo "Detected architecture: $Script:GoArch"
}

function Get-LinuxDistro {
    $Script:LinuxDistroId = ""
    $Script:LinuxDistroFamily = ""

    # Try /etc/os-release first (most modern distros)
    if (Test-Path "/etc/os-release") {
        $osRelease = Get-Content "/etc/os-release" -Raw
        # Use multiline regex to match ID= at start of line (not VERSION_ID, etc.)
        if ($osRelease -match '(?m)^ID=(.+)$') {
            $Script:LinuxDistroId = $Matches[1].Trim('"', "'", ' ')
        }
        if ($osRelease -match '(?m)^ID_LIKE=(.+)$') {
            $Script:LinuxDistroFamily = $Matches[1].Trim('"', "'", ' ')
        }
        else {
            $Script:LinuxDistroFamily = $Script:LinuxDistroId
        }
    }
    # Fallback to other release files
    elseif (Test-Path "/etc/debian_version") {
        $Script:LinuxDistroId = "debian"
        $Script:LinuxDistroFamily = "debian"
    }
    elseif (Test-Path "/etc/fedora-release") {
        $Script:LinuxDistroId = "fedora"
        $Script:LinuxDistroFamily = "fedora"
    }
    elseif (Test-Path "/etc/redhat-release") {
        $Script:LinuxDistroId = "rhel"
        $Script:LinuxDistroFamily = "rhel fedora"
    }
    elseif (Test-Path "/etc/arch-release") {
        $Script:LinuxDistroId = "arch"
        $Script:LinuxDistroFamily = "arch"
    }
    elseif (Test-Path "/etc/gentoo-release") {
        $Script:LinuxDistroId = "gentoo"
        $Script:LinuxDistroFamily = "gentoo"
    }
    elseif (Test-Path "/etc/alpine-release") {
        $Script:LinuxDistroId = "alpine"
        $Script:LinuxDistroFamily = "alpine"
    }
    elseif ((Test-Path "/etc/SuSE-release") -or (Test-Path "/etc/SUSE-brand")) {
        $Script:LinuxDistroId = "opensuse"
        $Script:LinuxDistroFamily = "suse"
    }

    $distroId = if ($Script:LinuxDistroId) { $Script:LinuxDistroId } else { "unknown" }
    $distroFamily = if ($Script:LinuxDistroFamily) { $Script:LinuxDistroFamily } else { "unknown" }
    Write-LogInfo "Detected Linux distribution: $distroId (family: $distroFamily)"
}

function Get-PackageManagerForDistro {
    # Check package managers typical for this distribution
    switch ($Script:LinuxDistroId) {
        # Debian-based
        { $_ -in @("debian", "ubuntu", "linuxmint", "pop", "elementary", "zorin", "kali", "raspbian", "neon") } {
            if (Test-CommandExists "apt-get") {
                $Script:PkgMgr = "apt"
                return $true
            }
        }
        # Fedora/RHEL-based
        { $_ -in @("fedora", "rhel", "centos", "rocky", "almalinux", "ol", "amzn") } {
            if (Test-CommandExists "dnf") {
                $Script:PkgMgr = "dnf"
                return $true
            }
        }
        # Arch-based
        { $_ -in @("arch", "manjaro", "endeavouros", "garuda", "artix") } {
            if (Test-CommandExists "pacman") {
                $Script:PkgMgr = "pacman"
                return $true
            }
        }
        # openSUSE
        { $_ -in @("opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles") } {
            if (Test-CommandExists "zypper") {
                $Script:PkgMgr = "zypper"
                return $true
            }
        }
        # Alpine
        "alpine" {
            if (Test-CommandExists "apk") {
                $Script:PkgMgr = "apk"
                return $true
            }
        }
        # Gentoo
        "gentoo" {
            if (Test-CommandExists "emerge") {
                $Script:PkgMgr = "emerge"
                return $true
            }
        }
        # Void Linux
        "void" {
            if (Test-CommandExists "xbps-install") {
                $Script:PkgMgr = "xbps-install"
                return $true
            }
        }
    }

    # Check by distro family (ID_LIKE)
    if ($Script:LinuxDistroFamily -match "debian|ubuntu") {
        if (Test-CommandExists "apt-get") {
            $Script:PkgMgr = "apt"
            return $true
        }
    }
    elseif ($Script:LinuxDistroFamily -match "fedora|rhel") {
        if (Test-CommandExists "dnf") {
            $Script:PkgMgr = "dnf"
            return $true
        }
    }
    elseif ($Script:LinuxDistroFamily -match "arch") {
        if (Test-CommandExists "pacman") {
            $Script:PkgMgr = "pacman"
            return $true
        }
    }
    elseif ($Script:LinuxDistroFamily -match "suse") {
        if (Test-CommandExists "zypper") {
            $Script:PkgMgr = "zypper"
            return $true
        }
    }

    return $false
}

function Get-PackageManagerFallback {
    # Try distribution-specific package managers in order of popularity
    if (Test-CommandExists "apt-get") {
        $Script:PkgMgr = "apt"
        return $true
    }
    elseif (Test-CommandExists "dnf") {
        $Script:PkgMgr = "dnf"
        return $true
    }
    elseif (Test-CommandExists "pacman") {
        $Script:PkgMgr = "pacman"
        return $true
    }
    elseif (Test-CommandExists "zypper") {
        $Script:PkgMgr = "zypper"
        return $true
    }
    elseif (Test-CommandExists "apk") {
        $Script:PkgMgr = "apk"
        return $true
    }
    elseif (Test-CommandExists "emerge") {
        $Script:PkgMgr = "emerge"
        return $true
    }
    elseif (Test-CommandExists "xbps-install") {
        $Script:PkgMgr = "xbps-install"
        return $true
    }

    # Cross-platform package managers as last resort
    Write-LogInfo "No distribution-specific package manager found, checking cross-platform options..."
    if (Test-CommandExists "snap") {
        $Script:PkgMgr = "snap"
        return $true
    }
    elseif (Test-CommandExists "flatpak") {
        $Script:PkgMgr = "flatpak"
        return $true
    }

    return $false
}

function Get-PackageManager {
    Write-LogInfo "Detecting package manager..."

    if ($Script:AutoarkPkgMgr) {
        $Script:PkgMgr = $Script:AutoarkPkgMgr
        Write-LogInfo "Using package manager from AUTARK_PKG_MGR: $Script:PkgMgr"
        return
    }

    switch ($Script:GoOs) {
        "windows" {
            if (Test-CommandExists "winget") {
                $Script:PkgMgr = "winget"
            }
            elseif (Test-CommandExists "choco") {
                $Script:PkgMgr = "choco"
            }
        }
        "darwin" {
            if (Test-CommandExists "brew") {
                $Script:PkgMgr = "brew"
            }
            elseif (Test-CommandExists "port") {
                $Script:PkgMgr = "port"
            }
        }
        "linux" {
            # First, detect the Linux distribution
            Get-LinuxDistro

            # Try to find package manager based on distribution
            if (-not (Get-PackageManagerForDistro)) {
                # Fallback: try all known package managers
                Get-PackageManagerFallback | Out-Null
            }
        }
    }

    if (-not $Script:PkgMgr) {
        Write-LogError "No supported package manager found."
        Write-LogError "Windows: winget, choco"
        Write-LogError "macOS: brew, port"
        Write-LogError "Linux (distribution-specific): apt, dnf, pacman, zypper, apk, emerge, xbps-install"
        Write-LogError "Linux (cross-platform): snap, flatpak"
        exit 1
    }

    Write-LogInfo "Detected package manager: $Script:PkgMgr"
}

# =============================================================================
# Phase 2: Install Required Tools
# =============================================================================

function Invoke-AsRoot {
    param([string]$Command, [string[]]$Arguments)

    # Check if we're already root
    $uid = & id -u 2>$null
    if ($uid -eq "0") {
        # Already root, run directly
        & $Command @Arguments
    }
    else {
        # Need sudo
        & sudo $Command @Arguments
    }
}

function Install-Package {
    param([string]$Package)

    Write-LogInfo "Installing $Package via $Script:PkgMgr..."

    switch ($Script:PkgMgr) {
        "winget" {
            & winget install --silent --accept-package-agreements --accept-source-agreements $Package
        }
        "choco" {
            & choco install -y $Package
        }
        "brew" {
            & brew install --quiet $Package
        }
        "port" {
            Invoke-AsRoot -Command "port" -Arguments @("install", $Package)
        }
        "apt" {
            Invoke-AsRoot -Command "apt-get" -Arguments @("update", "-qq")
            Invoke-AsRoot -Command "apt-get" -Arguments @("install", "-y", "-qq", $Package)
        }
        "dnf" {
            Invoke-AsRoot -Command "dnf" -Arguments @("install", "-y", "-q", $Package)
        }
        "pacman" {
            Invoke-AsRoot -Command "pacman" -Arguments @("-Sy", "--noconfirm", "--quiet", $Package)
        }
        "zypper" {
            Invoke-AsRoot -Command "zypper" -Arguments @("install", "-y", "-q", $Package)
        }
        "apk" {
            Invoke-AsRoot -Command "apk" -Arguments @("add", "--quiet", $Package)
        }
        "emerge" {
            Invoke-AsRoot -Command "emerge" -Arguments @("--quiet", $Package)
        }
        "xbps-install" {
            Invoke-AsRoot -Command "xbps-install" -Arguments @("-y", $Package)
        }
        "snap" {
            Invoke-AsRoot -Command "snap" -Arguments @("install", $Package)
        }
        "flatpak" {
            Invoke-AsRoot -Command "flatpak" -Arguments @("install", "-y", $Package)
        }
        default {
            Write-LogError "Unknown package manager: $Script:PkgMgr"
            exit 1
        }
    }
}

function Install-Git {
    Write-LogInfo "Checking for git..."

    if (Test-CommandExists "git") {
        Write-LogInfo "git is already installed."
        return
    }

    Write-LogInfo "git not found, installing..."
    Install-Package "git"

    # Refresh PATH for Windows
    if ($Script:GoOs -eq "windows") {
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    }

    if (-not (Test-CommandExists "git")) {
        Write-LogError "Failed to install git."
        exit 1
    }

    Write-LogInfo "git installed successfully."
}

function Install-Jq {
    Write-LogInfo "Checking for jq..."

    if (Test-CommandExists "jq") {
        Write-LogInfo "jq is already installed."
        return
    }

    Write-LogInfo "jq not found, installing..."
    Install-Package "jq"

    # Refresh PATH for Windows
    if ($Script:GoOs -eq "windows") {
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    }

    if (-not (Test-CommandExists "jq")) {
        Write-LogError "Failed to install jq."
        exit 1
    }

    Write-LogInfo "jq installed successfully."
}

# =============================================================================
# Phase 3: Download and Setup Golang
# =============================================================================

function Install-Golang {
    Write-LogInfo "Fetching Go version information..."

    $goJson = Invoke-RestMethod -Uri $Script:GoDownloadUrl -UseBasicParsing

    if (-not $goJson) {
        Write-LogError "Failed to fetch Go version information."
        exit 1
    }

    Write-LogInfo "Finding latest stable Go version for $Script:GoOs/$Script:GoArch..."

    $latestStable = $goJson | Where-Object { $_.stable -eq $true } | Select-Object -First 1

    if (-not $latestStable) {
        Write-LogError "No stable Go version found."
        exit 1
    }

    $archiveKind = "archive"
    $goFile = $latestStable.files | Where-Object {
        $_.os -eq $Script:GoOs -and
        $_.arch -eq $Script:GoArch -and
        $_.kind -eq $archiveKind
    } | Select-Object -First 1

    if (-not $goFile) {
        Write-LogError "No Go binary found for $Script:GoOs/$Script:GoArch."
        exit 1
    }

    $goFilename = $goFile.filename
    $goVersion = $goFile.version
    $goSha256 = $goFile.sha256

    Write-LogInfo "Latest stable Go version: $goVersion"
    Write-LogInfo "Filename: $goFilename"

    $goDownloadFullUrl = "https://go.dev/dl/$goFilename"
    $goArchivePath = Join-Path $Script:TempDir $goFilename

    Write-LogInfo "Downloading: $goDownloadFullUrl"
    Invoke-WebRequest -Uri $goDownloadFullUrl -OutFile $goArchivePath -UseBasicParsing

    Write-LogInfo "Verifying checksum..."
    $actualHash = (Get-FileHash -Path $goArchivePath -Algorithm SHA256).Hash.ToLower()

    if ($actualHash -ne $goSha256) {
        Write-LogError "Checksum verification failed!"
        Write-LogError "Expected: $goSha256"
        Write-LogError "Got: $actualHash"
        exit 1
    }
    Write-LogInfo "Checksum verified."

    Write-LogInfo "Extracting Go..."
    $goInstallDir = Join-Path $Script:TempDir "go"

    if ($Script:GoOs -eq "windows") {
        Expand-Archive -Path $goArchivePath -DestinationPath $Script:TempDir -Force
    }
    else {
        & tar -xzf $goArchivePath -C $Script:TempDir
    }

    if ($Script:GoOs -eq "windows") {
        $Script:GoBin = Join-Path $goInstallDir "bin\go.exe"
    }
    else {
        $Script:GoBin = Join-Path $goInstallDir "bin/go"
    }

    if (-not (Test-Path $Script:GoBin)) {
        Write-LogError "Go binary not found after extraction."
        exit 1
    }

    Write-LogInfo "Go $goVersion ready at: $Script:GoBin"
}

# =============================================================================
# Phase 4: Clone and Build Project
# =============================================================================

function Build-Autark {
    Write-LogInfo "Cloning repository: $Script:AutoarkRepoUrl"

    $projectDir = Join-Path $Script:TempDir "src"
    & git clone --depth 1 $Script:AutoarkRepoUrl $projectDir

    if ($LASTEXITCODE -ne 0) {
        Write-LogError "Failed to clone repository."
        exit 1
    }

    Write-LogInfo "Building project..."

    $env:GOROOT = Join-Path $Script:TempDir "go"

    $binaryName = "autark"
    if ($Script:GoOs -eq "windows") {
        $binaryName = "autark.exe"
    }

    $outputPath = Join-Path $Script:TempDir $binaryName

    Push-Location $projectDir
    try {
        # Download dependencies first
        Write-LogInfo "Downloading Go dependencies..."
        $modOutput = & $Script:GoBin mod download 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-LogError "Failed to download Go dependencies:"
            Write-Host $modOutput -ForegroundColor Red
            exit 1
        }

        # Build with error capture
        Write-LogInfo "Compiling..."
        $buildOutput = & $Script:GoBin build -o $outputPath . 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-LogError "Go build failed with error:"
            Write-Host $buildOutput -ForegroundColor Red
            exit 1
        }
    }
    finally {
        Pop-Location
    }

    if (-not (Test-Path $outputPath)) {
        Write-LogError "Build failed: binary not created."
        exit 1
    }

    Write-LogInfo "Build successful."
}

# =============================================================================
# Phase 5: Install Binary
# =============================================================================

function Install-Binary {
    Write-LogInfo "Determining installation directory..."

    $installDir = $null
    $binaryName = "autark"
    if ($Script:GoOs -eq "windows") {
        $binaryName = "autark.exe"
    }

    if ($Script:AutoarkBin) {
        $installDir = $Script:AutoarkBin
        Write-LogInfo "Using installation directory from AUTARK_BIN: $installDir"
    }
    else {
        if ($Script:GoOs -eq "windows") {
            $defaultDir = "C:\Program Files\autark"
        }
        else {
            $defaultDir = "/usr/local/bin"
        }

        if ([System.Environment]::UserInteractive) {
            $userInput = Read-Host "Enter installation directory [$defaultDir]"
            if ($userInput) {
                $installDir = $userInput
            }
            else {
                $installDir = $defaultDir
            }
        }
        else {
            $installDir = $defaultDir
            Write-LogInfo "Non-interactive mode, using default: $installDir"
        }
    }

    if (-not (Test-Path $installDir)) {
        Write-LogInfo "Creating directory: $installDir"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $sourceBinary = Join-Path $Script:TempDir $binaryName
    $destBinary = Join-Path $installDir $binaryName

    Write-LogInfo "Installing autark to $installDir..."
    Copy-Item -Path $sourceBinary -Destination $destBinary -Force

    if ($Script:GoOs -ne "windows") {
        & chmod 755 $destBinary
    }

    # Add to PATH for Windows if not already there
    if ($Script:GoOs -eq "windows") {
        $currentPath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
        if ($currentPath -notlike "*$installDir*") {
            Write-LogInfo "Adding $installDir to system PATH..."
            [System.Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "Machine")
            $env:Path = "$env:Path;$installDir"
        }
    }

    Write-LogSuccess "autark installed successfully to $destBinary"
}

# =============================================================================
# Main
# =============================================================================

function Main {
    Write-LogInfo "=== Autark Installation Script ==="
    Write-LogInfo ""

    try {
        # Phase 1: System Validation
        Test-AdminPrivileges
        Get-OperatingSystem
        Get-Architecture
        Get-PackageManager

        # Create temporary directory
        $Script:TempDir = Join-Path ([System.IO.Path]::GetTempPath()) "autark-install-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
        New-Item -ItemType Directory -Path $Script:TempDir -Force | Out-Null
        Write-LogInfo "Using temporary directory: $Script:TempDir"

        # Phase 2: Install Required Tools
        Install-Git
        Install-Jq

        # Phase 3: Download and Setup Golang
        Install-Golang

        # Phase 4: Clone and Build Project
        Build-Autark

        # Phase 5: Install Binary
        Install-Binary

        Write-LogInfo ""
        Write-LogSuccess "Installation complete!"
        Write-LogInfo "Run 'autark --help' to get started."
    }
    finally {
        # Phase 6: Cleanup
        Invoke-Cleanup
    }
}

# Run main function
Main
