package main

import (
	"log"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	log.Println("Starting audio player application")
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Println("Failed to create log file:", err)
	}
	defer f.Close()

	// Load configuration and populate fileList with audio tracks
	log.Println("Loading application configuration")
	config := LoadConfig()
	log.Printf("Configuration loaded, directories to scan: %v", config.Directories)
	updateTrackList(config.Directories)

	// Convert tracklist to list.Item slice for the UI
	listItems := make([]list.Item, len(trackList))
	for i, track := range trackList {
		listItems[i] = track
	}

	// Create the model with our listItems - no longer passing hardcoded tabs
	m := New(listItems)

	// Initialize and run the TUI
	log.Println("Starting TUI application")
	t := tea.NewProgram(*m, tea.WithAltScreen())

	if _, err := t.Run(); err != nil {
		log.Fatal("Error running application:", err)
	}

	// Clean up resources
	log.Println("Application shutting down, cleaning up resources")
	if player != nil {
		player.Close()
	}
	log.Println("Audio player application terminated")
}
