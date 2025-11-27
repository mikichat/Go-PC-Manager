package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("Starting Go PC Agent Installation...")

	// 1. Target Directory
	targetDir := `C:\Program Files\GoPCManager`
	exeName := "agent.exe"
	targetPath := filepath.Join(targetDir, exeName)

	// 2. Create Directory
	fmt.Printf("Creating directory: %s\n", targetDir)
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		fmt.Println("Please run as Administrator.")
		pause()
		os.Exit(1)
	}

	// 3. Copy Executable
	currentExe, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting current executable path: %v\n", err)
		pause()
		os.Exit(1)
	}
	currentDir := filepath.Dir(currentExe)
	sourcePath := filepath.Join(currentDir, exeName)

	fmt.Printf("Copying %s to %s\n", sourcePath, targetPath)
	err = copyFile(sourcePath, targetPath)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		fmt.Println("Make sure agent.exe is in the same directory as the installer.")
		pause()
		os.Exit(1)
	}

	// 4. Install Service
	fmt.Println("Installing Windows Service...")
	cmd := exec.Command(targetPath, "-service", "install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error installing service: %v\nOutput: %s\n", err, string(output))
		pause()
		os.Exit(1)
	}
	fmt.Println(string(output))

	// 5. Start Service
	fmt.Println("Starting Windows Service...")
	cmd = exec.Command(targetPath, "-service", "start")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error starting service: %v\nOutput: %s\n", err, string(output))
		pause()
		os.Exit(1)
	}
	fmt.Println(string(output))

	fmt.Println("\nInstallation Completed Successfully!")
	pause()
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func pause() {
	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
	time.Sleep(100 * time.Millisecond)
}
