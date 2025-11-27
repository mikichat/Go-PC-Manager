package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const AgentVersion = "1.0.0"

type VersionResponse struct {
	Version string `json:"version"`
}

func checkForUpdates(serverAddr string) {
	log.Println("Checking for updates...")
	resp, err := http.Get(fmt.Sprintf("http://%s/version", serverAddr))
	if err != nil {
		log.Printf("Failed to check version: %v", err)
		return
	}
	defer resp.Body.Close()

	var versionResp VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		log.Printf("Failed to decode version response: %v", err)
		return
	}

	if versionResp.Version != AgentVersion {
		log.Printf("New version available: %s (current: %s)", versionResp.Version, AgentVersion)
		doUpdate(serverAddr)
	} else {
		log.Println("Agent is up to date.")
	}
}

func doUpdate(serverAddr string) {
	log.Println("Starting update process...")

	// 1. Download new executable
	resp, err := http.Get(fmt.Sprintf("http://%s/updates/agent.exe", serverAddr))
	if err != nil {
		log.Printf("Failed to download update: %v", err)
		return
	}
	defer resp.Body.Close()

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return
	}

	newExePath := exePath + ".new"
	out, err := os.Create(newExePath)
	if err != nil {
		log.Printf("Failed to create new executable file: %v", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Printf("Failed to write new executable file: %v", err)
		return
	}
	out.Close() // Ensure file is closed before renaming

	// 2. Rename current executable to .old
	oldExePath := exePath + ".old"
	// Remove old backup if exists
	os.Remove(oldExePath)

	err = os.Rename(exePath, oldExePath)
	if err != nil {
		log.Printf("Failed to rename current executable: %v", err)
		return
	}

	// 3. Rename new executable to current name
	err = os.Rename(newExePath, exePath)
	if err != nil {
		log.Printf("Failed to rename new executable: %v", err)
		// Try to rollback
		os.Rename(oldExePath, exePath)
		return
	}

	log.Println("Update downloaded and installed. Restarting service...")
	// Service manager (Windows Service) should handle restart if we exit
	// But simply exiting might be interpreted as failure.
	// For now, we will just exit and let the service recovery options (if configured) or manual restart handle it.
	// Ideally, we should trigger a service restart command.
	os.Exit(0)
}
