package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type track struct {
	title       string
	description string
	file        *os.File
	filename    string
	audioformat audioType
}

// Required by the list.Item interface
func (track track) Title() string       { return track.title }
func (track track) FilterValue() string { return track.title }
func (track track) Description() string { return track.description }

// updateTrackList walks through the given directories and finds audio files
// Audio files are added to the global trackList
func updateTrackList(directories []string) {
	log.Printf("Scanning directories for audio files: %v", directories)
	for _, dir := range directories {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Printf("Error accessing path %s: %v\n", path, err)
				return nil // Continue walking despite errors
			}

			// Skip directories
			if d.IsDir() {
				return nil
			}

			// Check if the file is an mp3
			if filepath.Ext(d.Name()) == ".mp3" {
				log.Printf("Found audio file: %s", path)

				// Open the file
				file, err := os.Open(path)
				if err != nil {
					log.Printf("Error opening file %s: %v\n", path, err)
				}

				// Get file metadata
				title, description, audioformat := TitleDescriptionAudioFormat(file)
				log.Printf("Processed audio file: Title=%s, Description=%s, Format=%v", title, description, audioformat)

				// Create a track struct
				t := track{
					filename:    path,
					file:        file,
					title:       title,
					description: description,
					audioformat: audioformat,
				}

				// Add to the trackList
				trackList = append(trackList, t)
			}
			return nil
		})
	}
	log.Printf("Found %d audio tracks in total", len(trackList))
}
