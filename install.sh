#!/bin/sh
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

set -e

# =============================================================================
# Configuration
# =============================================================================

AUTARK_REPO_URL="${AUTARK_REPO_URL:-https://github.com/mkloubert/autark.git}"
AUTARK_PKG_MGR="${AUTARK_PKG_MGR:-}"
AUTARK_BIN="${AUTARK_BIN:-}"
GO_DOWNLOAD_URL="https://go.dev/dl/?mode=json"

# =============================================================================
# Utility Functions
# =============================================================================

log_info() {
    printf "[INFO] %s\n" "$1"
}

log_error() {
    printf "[ERROR] %s\n" "$1" >&2
}

log_success() {
    printf "[SUCCESS] %s\n" "$1"
}

cleanup() {
    if [ -n "${TEMP_DIR:-}" ] && [ -d "$TEMP_DIR" ]; then
        log_info "Cleaning up temporary directory..."
        rm -rf "$TEMP_DIR"
    fi
}

trap cleanup EXIT

# =============================================================================
# Phase 1: System Validation and Prerequisites
# =============================================================================

check_root() {
    log_info "Checking for root/admin privileges..."
    if [ "$(id -u)" -ne 0 ]; then
        log_error "This script must be run as root or with sudo."
        exit 1
    fi
    log_info "Root privileges confirmed."
}

detect_os() {
    log_info "Detecting operating system..."
    OS_TYPE=""
    case "$(uname -s)" in
        Linux*)
            OS_TYPE="linux"
            ;;
        Darwin*)
            OS_TYPE="darwin"
            ;;
        FreeBSD*)
            OS_TYPE="freebsd"
            ;;
        NetBSD*)
            OS_TYPE="netbsd"
            ;;
        OpenBSD*)
            OS_TYPE="openbsd"
            ;;
        DragonFly*)
            OS_TYPE="dragonfly"
            ;;
        SunOS*)
            if [ -f /etc/illumos-release ]; then
                OS_TYPE="illumos"
            else
                OS_TYPE="solaris"
            fi
            ;;
        AIX*)
            OS_TYPE="aix"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    log_info "Detected OS: $OS_TYPE"
}

detect_arch() {
    log_info "Detecting processor architecture..."
    ARCH_TYPE=""
    case "$(uname -m)" in
        x86_64|amd64)
            ARCH_TYPE="amd64"
            ;;
        aarch64|arm64)
            ARCH_TYPE="arm64"
            ;;
        ppc64le)
            ARCH_TYPE="ppc64le"
            ;;
        ppc64)
            ARCH_TYPE="ppc64"
            ;;
        s390x)
            ARCH_TYPE="s390x"
            ;;
        riscv64)
            ARCH_TYPE="riscv64"
            ;;
        mips64)
            ARCH_TYPE="mips64"
            ;;
        mips64el|mips64le)
            ARCH_TYPE="mips64le"
            ;;
        loongarch64|loong64)
            ARCH_TYPE="loong64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m). Only 64-bit systems are supported."
            exit 1
            ;;
    esac
    log_info "Detected architecture: $ARCH_TYPE"
}

detect_linux_distro() {
    LINUX_DISTRO=""
    LINUX_DISTRO_ID=""

    # Try /etc/os-release first (most modern distros)
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        LINUX_DISTRO_ID="${ID:-}"
        LINUX_DISTRO="${ID_LIKE:-$LINUX_DISTRO_ID}"
    # Fallback to other release files
    elif [ -f /etc/debian_version ]; then
        LINUX_DISTRO_ID="debian"
        LINUX_DISTRO="debian"
    elif [ -f /etc/fedora-release ]; then
        LINUX_DISTRO_ID="fedora"
        LINUX_DISTRO="fedora"
    elif [ -f /etc/redhat-release ]; then
        LINUX_DISTRO_ID="rhel"
        LINUX_DISTRO="rhel fedora"
    elif [ -f /etc/arch-release ]; then
        LINUX_DISTRO_ID="arch"
        LINUX_DISTRO="arch"
    elif [ -f /etc/gentoo-release ]; then
        LINUX_DISTRO_ID="gentoo"
        LINUX_DISTRO="gentoo"
    elif [ -f /etc/alpine-release ]; then
        LINUX_DISTRO_ID="alpine"
        LINUX_DISTRO="alpine"
    elif [ -f /etc/SuSE-release ] || [ -f /etc/SUSE-brand ]; then
        LINUX_DISTRO_ID="opensuse"
        LINUX_DISTRO="suse"
    fi

    log_info "Detected Linux distribution: ${LINUX_DISTRO_ID:-unknown} (family: ${LINUX_DISTRO:-unknown})"
}

detect_package_manager_for_distro() {
    # Check package managers typical for this distribution
    case "$LINUX_DISTRO_ID" in
        # Debian-based
        debian|ubuntu|linuxmint|pop|elementary|zorin|kali|raspbian|neon)
            if command -v apt-get >/dev/null 2>&1; then
                PKG_MGR="apt"
                return 0
            fi
            ;;
        # Fedora/RHEL-based
        fedora|rhel|centos|rocky|almalinux|ol|amzn)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MGR="dnf"
                return 0
            fi
            ;;
        # Arch-based
        arch|manjaro|endeavouros|garuda|artix)
            if command -v pacman >/dev/null 2>&1; then
                PKG_MGR="pacman"
                return 0
            fi
            ;;
        # openSUSE
        opensuse|opensuse-leap|opensuse-tumbleweed|sles)
            if command -v zypper >/dev/null 2>&1; then
                PKG_MGR="zypper"
                return 0
            fi
            ;;
        # Alpine
        alpine)
            if command -v apk >/dev/null 2>&1; then
                PKG_MGR="apk"
                return 0
            fi
            ;;
        # Gentoo
        gentoo)
            if command -v emerge >/dev/null 2>&1; then
                PKG_MGR="emerge"
                return 0
            fi
            ;;
        # Void Linux
        void)
            if command -v xbps-install >/dev/null 2>&1; then
                PKG_MGR="xbps-install"
                return 0
            fi
            ;;
    esac

    # Check by distro family (ID_LIKE)
    case "$LINUX_DISTRO" in
        *debian*|*ubuntu*)
            if command -v apt-get >/dev/null 2>&1; then
                PKG_MGR="apt"
                return 0
            fi
            ;;
        *fedora*|*rhel*)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MGR="dnf"
                return 0
            fi
            ;;
        *arch*)
            if command -v pacman >/dev/null 2>&1; then
                PKG_MGR="pacman"
                return 0
            fi
            ;;
        *suse*)
            if command -v zypper >/dev/null 2>&1; then
                PKG_MGR="zypper"
                return 0
            fi
            ;;
    esac

    return 1
}

detect_package_manager_fallback() {
    # Try distribution-specific package managers in order of popularity
    if command -v apt-get >/dev/null 2>&1; then
        PKG_MGR="apt"
    elif command -v dnf >/dev/null 2>&1; then
        PKG_MGR="dnf"
    elif command -v pacman >/dev/null 2>&1; then
        PKG_MGR="pacman"
    elif command -v zypper >/dev/null 2>&1; then
        PKG_MGR="zypper"
    elif command -v apk >/dev/null 2>&1; then
        PKG_MGR="apk"
    elif command -v emerge >/dev/null 2>&1; then
        PKG_MGR="emerge"
    elif command -v xbps-install >/dev/null 2>&1; then
        PKG_MGR="xbps-install"
    fi

    if [ -n "$PKG_MGR" ]; then
        return 0
    fi

    # Cross-platform package managers as last resort
    log_info "No distribution-specific package manager found, checking cross-platform options..."
    if command -v snap >/dev/null 2>&1; then
        PKG_MGR="snap"
    elif command -v flatpak >/dev/null 2>&1; then
        PKG_MGR="flatpak"
    fi

    if [ -n "$PKG_MGR" ]; then
        return 0
    fi

    return 1
}

detect_package_manager() {
    log_info "Detecting package manager..."

    if [ -n "$AUTARK_PKG_MGR" ]; then
        PKG_MGR="$AUTARK_PKG_MGR"
        log_info "Using package manager from AUTARK_PKG_MGR: $PKG_MGR"
        return
    fi

    PKG_MGR=""

    case "$OS_TYPE" in
        linux)
            # First, detect the Linux distribution
            detect_linux_distro

            # Try to find package manager based on distribution
            if ! detect_package_manager_for_distro; then
                # Fallback: try all known package managers
                detect_package_manager_fallback
            fi
            ;;
        darwin)
            if command -v brew >/dev/null 2>&1; then
                PKG_MGR="brew"
            elif command -v port >/dev/null 2>&1; then
                PKG_MGR="port"
            fi
            ;;
        freebsd|netbsd|openbsd|dragonfly)
            if command -v pkg >/dev/null 2>&1; then
                PKG_MGR="pkg"
            fi
            ;;
        *)
            PKG_MGR=""
            ;;
    esac

    if [ -z "$PKG_MGR" ]; then
        log_error "No supported package manager found."
        log_error "Distribution-specific: apt, dnf, pacman, zypper, apk, emerge, xbps-install"
        log_error "Cross-platform: snap, flatpak"
        log_error "macOS: brew, port"
        exit 1
    fi

    log_info "Detected package manager: $PKG_MGR"
}

# =============================================================================
# Phase 2: Install Required Tools
# =============================================================================

pkg_install() {
    PACKAGE="$1"
    log_info "Installing $PACKAGE via $PKG_MGR..."

    case "$PKG_MGR" in
        apt)
            apt-get update -qq
            apt-get install -y -qq "$PACKAGE"
            ;;
        dnf)
            dnf install -y -q "$PACKAGE"
            ;;
        pacman)
            pacman -Sy --noconfirm --quiet "$PACKAGE"
            ;;
        zypper)
            zypper install -y -q "$PACKAGE"
            ;;
        apk)
            apk add --quiet "$PACKAGE"
            ;;
        emerge)
            emerge --quiet "$PACKAGE"
            ;;
        xbps-install)
            xbps-install -y "$PACKAGE"
            ;;
        snap)
            snap install "$PACKAGE"
            ;;
        flatpak)
            flatpak install -y "$PACKAGE"
            ;;
        brew)
            brew install --quiet "$PACKAGE"
            ;;
        port)
            port install "$PACKAGE"
            ;;
        pkg)
            pkg install -y "$PACKAGE"
            ;;
        *)
            log_error "Unknown package manager: $PKG_MGR"
            exit 1
            ;;
    esac
}

install_git() {
    log_info "Checking for git..."
    if command -v git >/dev/null 2>&1; then
        log_info "git is already installed."
        return
    fi

    log_info "git not found, installing..."
    pkg_install git

    if ! command -v git >/dev/null 2>&1; then
        log_error "Failed to install git."
        exit 1
    fi
    log_info "git installed successfully."
}

install_downloader() {
    log_info "Checking for curl or wget..."

    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
        log_info "Using curl for downloads."
        return
    fi

    if command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
        log_info "Using wget for downloads."
        return
    fi

    log_info "Neither curl nor wget found, installing curl..."
    pkg_install curl

    if ! command -v curl >/dev/null 2>&1; then
        log_error "Failed to install curl."
        exit 1
    fi
    DOWNLOADER="curl"
    log_info "curl installed successfully."
}

install_jq() {
    log_info "Checking for jq..."
    if command -v jq >/dev/null 2>&1; then
        log_info "jq is already installed."
        return
    fi

    log_info "jq not found, installing..."
    pkg_install jq

    if ! command -v jq >/dev/null 2>&1; then
        log_error "Failed to install jq."
        exit 1
    fi
    log_info "jq installed successfully."
}

install_tar() {
    log_info "Checking for tar..."
    if command -v tar >/dev/null 2>&1; then
        log_info "tar is already installed."
        return
    fi

    log_info "tar not found, installing..."
    pkg_install tar

    if ! command -v tar >/dev/null 2>&1; then
        log_error "Failed to install tar."
        exit 1
    fi
    log_info "tar installed successfully."
}

# =============================================================================
# Phase 3: Download and Setup Golang
# =============================================================================

download_file() {
    URL="$1"
    OUTPUT="$2"

    log_info "Downloading: $URL"

    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL -o "$OUTPUT" "$URL"
    else
        wget -q -O "$OUTPUT" "$URL"
    fi
}

download_to_stdout() {
    URL="$1"

    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$URL"
    else
        wget -q -O- "$URL"
    fi
}

setup_golang() {
    log_info "Fetching Go version information..."

    GO_JSON=$(download_to_stdout "$GO_DOWNLOAD_URL")

    if [ -z "$GO_JSON" ]; then
        log_error "Failed to fetch Go version information."
        exit 1
    fi

    log_info "Finding latest stable Go version for $OS_TYPE/$ARCH_TYPE..."

    GO_FILE_INFO=$(echo "$GO_JSON" | jq -r --arg os "$OS_TYPE" --arg arch "$ARCH_TYPE" '
        [.[] | select(.stable == true)] | first |
        .files[] | select(.os == $os and .arch == $arch and .kind == "archive")
    ' 2>/dev/null)

    if [ -z "$GO_FILE_INFO" ] || [ "$GO_FILE_INFO" = "null" ]; then
        log_error "No Go binary found for $OS_TYPE/$ARCH_TYPE."
        exit 1
    fi

    GO_FILENAME=$(echo "$GO_FILE_INFO" | jq -r '.filename')
    GO_VERSION=$(echo "$GO_FILE_INFO" | jq -r '.version')
    GO_SHA256=$(echo "$GO_FILE_INFO" | jq -r '.sha256')

    log_info "Latest stable Go version: $GO_VERSION"
    log_info "Filename: $GO_FILENAME"

    GO_DOWNLOAD_FULL_URL="https://go.dev/dl/$GO_FILENAME"
    GO_ARCHIVE_PATH="$TEMP_DIR/$GO_FILENAME"

    download_file "$GO_DOWNLOAD_FULL_URL" "$GO_ARCHIVE_PATH"

    log_info "Verifying checksum..."
    ACTUAL_SHA256=""
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_SHA256=$(sha256sum "$GO_ARCHIVE_PATH" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_SHA256=$(shasum -a 256 "$GO_ARCHIVE_PATH" | cut -d' ' -f1)
    else
        log_info "Warning: sha256sum not available, skipping checksum verification."
        ACTUAL_SHA256="$GO_SHA256"
    fi

    if [ "$ACTUAL_SHA256" != "$GO_SHA256" ]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $GO_SHA256"
        log_error "Got: $ACTUAL_SHA256"
        exit 1
    fi
    log_info "Checksum verified."

    log_info "Extracting Go..."
    GO_INSTALL_DIR="$TEMP_DIR/go"
    mkdir -p "$GO_INSTALL_DIR"
    tar -xzf "$GO_ARCHIVE_PATH" -C "$TEMP_DIR"

    GO_BIN="$TEMP_DIR/go/bin/go"
    if [ ! -x "$GO_BIN" ]; then
        log_error "Go binary not found after extraction."
        exit 1
    fi

    log_info "Go $GO_VERSION ready at: $GO_BIN"
}

# =============================================================================
# Phase 4: Clone and Build Project
# =============================================================================

clone_and_build() {
    log_info "Cloning repository: $AUTARK_REPO_URL"

    PROJECT_DIR="$TEMP_DIR/src"
    git clone --depth 1 "$AUTARK_REPO_URL" "$PROJECT_DIR"

    log_info "Building project..."
    cd "$PROJECT_DIR"

    export GOROOT="$TEMP_DIR/go"
    export PATH="$TEMP_DIR/go/bin:$PATH"

    # Download dependencies first
    log_info "Downloading Go dependencies..."
    if ! "$GO_BIN" mod download 2>&1; then
        log_error "Failed to download Go dependencies."
        exit 1
    fi

    # Build the project
    log_info "Compiling..."
    if ! "$GO_BIN" build -o "$TEMP_DIR/autark" . 2>&1; then
        log_error "Go build failed."
        exit 1
    fi

    if [ ! -f "$TEMP_DIR/autark" ]; then
        log_error "Build failed: binary not created."
        exit 1
    fi

    log_info "Build successful."
}

# =============================================================================
# Phase 5: Install Binary
# =============================================================================

install_binary() {
    log_info "Determining installation directory..."

    INSTALL_DIR=""

    if [ -n "$AUTARK_BIN" ]; then
        INSTALL_DIR="$AUTARK_BIN"
        log_info "Using installation directory from AUTARK_BIN: $INSTALL_DIR"
    else
        DEFAULT_DIR="/usr/local/bin"

        if [ -t 0 ]; then
            printf "Enter installation directory [%s]: " "$DEFAULT_DIR"
            read -r USER_INPUT
            if [ -n "$USER_INPUT" ]; then
                INSTALL_DIR="$USER_INPUT"
            else
                INSTALL_DIR="$DEFAULT_DIR"
            fi
        else
            INSTALL_DIR="$DEFAULT_DIR"
            log_info "Non-interactive mode, using default: $INSTALL_DIR"
        fi
    fi

    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    log_info "Installing autark to $INSTALL_DIR..."
    cp "$TEMP_DIR/autark" "$INSTALL_DIR/autark"
    chmod 755 "$INSTALL_DIR/autark"

    log_success "autark installed successfully to $INSTALL_DIR/autark"
}

# =============================================================================
# Main
# =============================================================================

main() {
    log_info "=== Autark Installation Script ==="
    log_info ""

    # Phase 1: System Validation
    check_root
    detect_os
    detect_arch
    detect_package_manager

    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    log_info "Using temporary directory: $TEMP_DIR"

    # Phase 2: Install Required Tools
    install_git
    install_downloader
    install_jq
    install_tar

    # Phase 3: Download and Setup Golang
    setup_golang

    # Phase 4: Clone and Build Project
    clone_and_build

    # Phase 5: Install Binary
    install_binary

    # Phase 6: Cleanup (handled by trap)
    log_info ""
    log_success "Installation complete!"
    log_info "Run 'autark --help' to get started."
}

main "$@"
