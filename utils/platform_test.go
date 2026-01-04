// The MIT License (MIT)
// Copyright (c) 2026 Marcel Joachim Kloubert <https://marcel.coffee>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package utils

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform := DetectPlatform()

	if platform == nil {
		t.Fatal("DetectPlatform returned nil")
	}

	// Verify OS detection
	switch runtime.GOOS {
	case "linux":
		if platform.OS != OSLinux {
			t.Errorf("Expected OSLinux, got %s", platform.OS)
		}
	case "darwin":
		if platform.OS != OSDarwin {
			t.Errorf("Expected OSDarwin, got %s", platform.OS)
		}
	case "windows":
		if platform.OS != OSWindows {
			t.Errorf("Expected OSWindows, got %s", platform.OS)
		}
	case "freebsd":
		if platform.OS != OSFreeBSD {
			t.Errorf("Expected OSFreeBSD, got %s", platform.OS)
		}
	}

	// Verify architecture detection
	if platform.Arch != runtime.GOARCH {
		t.Errorf("Expected arch %s, got %s", runtime.GOARCH, platform.Arch)
	}
}

func TestOSTypeConstants(t *testing.T) {
	tests := []struct {
		os       OSType
		expected string
	}{
		{OSLinux, "linux"},
		{OSDarwin, "darwin"},
		{OSWindows, "windows"},
		{OSFreeBSD, "freebsd"},
		{OSUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.os) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.os))
		}
	}
}

func TestLinuxDistroConstants(t *testing.T) {
	tests := []struct {
		distro   LinuxDistro
		expected string
	}{
		{DistroDebian, "debian"},
		{DistroUbuntu, "ubuntu"},
		{DistroFedora, "fedora"},
		{DistroRHEL, "rhel"},
		{DistroCentOS, "centos"},
		{DistroArch, "arch"},
		{DistroAlpine, "alpine"},
		{DistroOpenSUSE, "opensuse"},
		{DistroGentoo, "gentoo"},
		{DistroVoid, "void"},
		{DistroUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.distro) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.distro))
		}
	}
}

func TestPackageManagerConstants(t *testing.T) {
	tests := []struct {
		pkgMgr   PackageManager
		expected string
	}{
		{PkgMgrApt, "apt"},
		{PkgMgrDnf, "dnf"},
		{PkgMgrPacman, "pacman"},
		{PkgMgrApk, "apk"},
		{PkgMgrZypper, "zypper"},
		{PkgMgrEmerge, "emerge"},
		{PkgMgrXbpsInstall, "xbps-install"},
		{PkgMgrSnap, "snap"},
		{PkgMgrFlatpak, "flatpak"},
		{PkgMgrBrew, "brew"},
		{PkgMgrPort, "port"},
		{PkgMgrPkg, "pkg"},
		{PkgMgrChoco, "choco"},
		{PkgMgrWinget, "winget"},
		{PkgMgrUnknown, "unknown"},
	}

	for _, tt := range tests {
		if string(tt.pkgMgr) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.pkgMgr))
		}
	}
}

func TestPlatformInfo_LinuxDistroDetection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	platform := DetectPlatform()

	// On Linux, we should have detected a distro
	if platform.LinuxDistro == "" {
		t.Log("Warning: Linux distro not detected (might be unsupported distro)")
	}
}

func TestPlatformInfo_PackageManagerDetection(t *testing.T) {
	platform := DetectPlatform()

	// On most systems, we should detect some package manager
	// This is not a hard failure as some minimal systems might not have one
	if platform.PackageManager == PkgMgrUnknown {
		t.Log("Warning: No package manager detected")
	}
}
