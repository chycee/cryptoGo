package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

const (
	AppName = "crypto-go"
)

// GetWorkspaceDir returns the root directory for all runtime data.
// It prioritizes a local "_workspace" directory if it exists (Portable/Dev mode).
// Otherwise, it returns the OS-standard data directory.
func GetWorkspaceDir() string {
	// 1. Check for local workspace (Priority 1: Portable/Dev)
	localDir := "_workspace"
	if _, err := os.Stat(localDir); err == nil {
		return localDir
	}

	// 2. Identify OS Standard Data Dir (Priority 2: Production)
	var baseDir string
	switch runtime.GOOS {
	case "windows":
		// Windows: %AppData%\crypto-go
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		// macOS: ~/Library/Application Support/crypto-go
		home, _ := os.UserHomeDir()
		baseDir = filepath.Join(home, "Library", "Application Support")
	case "linux":
		// Linux: ~/.local/share/crypto-go (XDG_DATA_HOME)
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome != "" {
			baseDir = dataHome
		} else {
			home, _ := os.UserHomeDir()
			baseDir = filepath.Join(home, ".local", "share")
		}
	default:
		// Fallback to local
		return localDir
	}

	res := filepath.Join(baseDir, AppName)
	return res
}

// EnsureDir creates the directory if it doesn't exist with safe permissions (0755).
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// CreateLockFile attempts to lock a file to prevent multiple instances.
// Uses OS-level flock which auto-releases when process exits (even on crash).
func CreateLockFile(workDir string) (func(), error) {
	lockPath := filepath.Join(workDir, "instance.lock")

	// Ensure parent dir exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	// Try non-blocking exclusive lock
	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("another instance is already running (lock: %s)", lockPath)
	}

	// Write current PID for debugging
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(fmt.Sprintf("%d", os.Getpid()))

	// Keep file open — lock is held as long as the file descriptor is open
	closer := func() {
		f.Close()
		os.Remove(lockPath)
	}

	return closer, nil
}

// lockFile attempts to acquire an exclusive, non-blocking lock on the given file.
// It uses syscall.Flock for OS-level file locking.
func lockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// ResolveConfigPath attempts to find the config.yaml.
// Priority: 1. Current Dir, 2. OS Config Dir
func ResolveConfigPath() string {
	defaultPath := filepath.Join("configs", "config.yaml")

	// 1. Current working directory (standard)
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	// 2. OS Standard Config Dir
	configRoot, err := os.UserConfigDir()
	if err == nil {
		osPath := filepath.Join(configRoot, AppName, "config.yaml")
		if _, err := os.Stat(osPath); err == nil {
			return osPath
		}
	}

	// Return default and let LoadConfig handle the "file not found" error if it's really missing
	return defaultPath
}
