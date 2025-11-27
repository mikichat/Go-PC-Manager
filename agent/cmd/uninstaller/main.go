package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("Starting Go PC Agent Uninstallation...")

	targetDir := `C:\Program Files\GoPCManager`
	exeName := "agent.exe"
	targetPath := filepath.Join(targetDir, exeName)

	// 1. Stop Service
	fmt.Println("Stopping Windows Service...")
	cmd := exec.Command(targetPath, "-service", "stop")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 서비스가 이미 중지되었거나 없을 수 있으므로 경고만 출력
		fmt.Printf("Warning stopping service: %v\nOutput: %s\n", err, string(output))
	} else {
		fmt.Println(string(output))
	}

	// 2. Uninstall Service
	fmt.Println("Uninstalling Windows Service...")
	cmd = exec.Command(targetPath, "-service", "uninstall")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning uninstalling service: %v\nOutput: %s\n", err, string(output))
	} else {
		fmt.Println(string(output))
	}

	// 3. Remove Files
	fmt.Println("Removing files...")
	err = os.Remove(targetPath)
	if err != nil {
		fmt.Printf("Error removing agent.exe: %v\n", err)
		fmt.Println("You may need to manually delete the file.")
	} else {
		fmt.Println("agent.exe removed.")
	}

	// 4. Remove Directory
	err = os.Remove(targetDir)
	if err != nil {
		fmt.Printf("Error removing directory: %v\n", err)
	} else {
		fmt.Println("Directory removed.")
	}

	fmt.Println("\nUninstallation Completed!")
	pause()
}

func pause() {
	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
	time.Sleep(100 * time.Millisecond)
}
