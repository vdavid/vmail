package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runCommand executes a command and optionally captures its output.
func runCommand(cmd *exec.Cmd, captureOutput bool) (string, error) {
	var stdout, stderr bytes.Buffer
	if captureOutput {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}
	return output, err
}

// commandExists checks if a command exists in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// getGoPath returns the GOPATH environment variable.
func getGoPath() string {
	cmd := exec.Command("go", "env", "GOPATH")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ensureToolInstalled ensures a Go tool is installed, installing it if necessary.
func ensureToolInstalled(toolName, installCmd string) error {
	if commandExists(toolName) {
		return nil
	}

	// Check in GOPATH/bin
	gopath := getGoPath()
	if gopath != "" {
		toolPath := filepath.Join(gopath, "bin", toolName)
		if _, err := os.Stat(toolPath); err == nil {
			return nil
		}
	}

	fmt.Printf("%sInstalling %s...%s\n", colorYellow, toolName, colorReset)
	parts := strings.Fields(installCmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// addGoPathToPath adds GOPATH/bin to PATH if not already present.
func addGoPathToPath() {
	gopath := getGoPath()
	if gopath == "" {
		return
	}
	gopathBin := filepath.Join(gopath, "bin")
	path := os.Getenv("PATH")
	if !strings.Contains(path, gopathBin) {
		err := os.Setenv("PATH", gopathBin+string(os.PathListSeparator)+path)
		if err != nil {
			fmt.Printf("Warning: Failed to add %s to PATH: %v\n", gopathBin, err)
			return
		}
	}
}

// findRootDir finds the project root directory by looking for backend/go.mod and frontend/package.json.
func findRootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if this is the project root by looking for backend/go.mod and frontend/package.json
		backendGoMod := filepath.Join(dir, "backend", "go.mod")
		frontendPackageJson := filepath.Join(dir, "frontend", "package.json")
		if _, err := os.Stat(backendGoMod); err == nil {
			if _, err := os.Stat(frontendPackageJson); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (looking for backend/go.mod and frontend/package.json)")
		}
		dir = parent
	}
}
