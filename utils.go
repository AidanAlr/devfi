package main

import (
	"log"
	"os"

	tag "github.com/dhowden/tag"
)

func TitleDescriptionAudioFormat(file *os.File) (title string, description string, audioformat audioType) {
	log.Printf("Reading metadata for file: %s", file.Name())
	tr, err := tag.ReadFrom(file)
	if err != nil {
		log.Printf("Error reading metadata: %v", err)
	}

	switch tr.FileType() {
	case "MP3":
		audioformat = typeMP3
	}

	trackTitle := tr.Title()
	artist := tr.Artist()
	album := tr.Album()
	if len(album) == 0 {
		album = "Unknown Album"
	}
	if len(artist) == 0 {
		artist = "Unknown Artist"
	}
	// Create a description combining album and artist
	trackDescription := "* " + album + " ⊚ " + artist + " *"
	// If the track metadata does not contain a title, use the filename
	if len(trackTitle) == 0 {
		trackTitle = file.Name()
		log.Println("No title in metadata, using filename instead")
	}

	if len(trackDescription) == 3 {
		trackDescription = "No description available"
		log.Println("No album/artist in metadata, using default description")
	}

	return trackTitle, trackDescription, audioformat
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
