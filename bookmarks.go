package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// BookmarkEntry represents a single bookmark with a timestamp
type BookmarkEntry string

func (be BookmarkEntry) String() string {
	return string(be)
}

// BookmarkMap maps filepaths to a list of bookmarks
type BookmarkMap map[string][]BookmarkEntry

// BookmarksManager handles loading and saving of bookmarks
type BookmarksManager struct {
	Bookmarks BookmarkMap
	filePath  string
}

// NewBookmarksManager creates a new bookmarks manager
func NewBookmarksManager() *BookmarksManager {
	bm := &BookmarksManager{
		Bookmarks: make(BookmarkMap),
		filePath:  path.Join(getUserHomeDir(), ".bookmarks"),
	}

	// Load existing bookmarks
	if err := bm.LoadBookmarks(); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create it immediately
			file, createErr := os.Create(bm.filePath)
			if createErr != nil {
				log.Printf("Error creating bookmarks file: %v", createErr)
			} else {
				file.Close()
				log.Printf("Created new bookmarks file at %s", bm.filePath)
			}
		} else {
			// Some other error occurred when loading
			log.Printf("Error loading bookmarks: %v", err)
		}
	}

	return bm
}

func (bm *BookmarksManager) ConvertDurationToBookmark(duration time.Duration) BookmarkEntry {
	// Convert the duration to total seconds
	totalSeconds := int(duration.Seconds())

	// Calculate hours, minutes, and seconds
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	// Format as "hh:mm:ss"
	return BookmarkEntry(fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))
}

// LoadBookmarks loads the bookmarks from the JSON file
func (bm *BookmarksManager) LoadBookmarks() error {
	// Check if file exists
	if _, err := os.Stat(bm.filePath); os.IsNotExist(err) {
		return err
	}

	content, err := os.ReadFile(bm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read bookmarks file: %w", err)
	}

	if len(content) == 0 {
		// Empty file, initialize empty bookmarks
		bm.Bookmarks = make(BookmarkMap)
		return nil
	}

	return json.Unmarshal(content, &bm.Bookmarks)
}

// SaveBookmarks saves the current bookmarks to the JSON file
func (bm *BookmarksManager) SaveBookmarks() error {
	content, err := json.MarshalIndent(bm.Bookmarks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bookmarks: %w", err)
	}
	return os.WriteFile(bm.filePath, content, 0o644)
}

// AddBookmark adds a bookmark for a specific file
func (bm *BookmarksManager) AddBookmark(filePath string, bookmark BookmarkEntry) (index int) {
	// Create slice for this filepath if it doesn't exist
	if _, exists := bm.Bookmarks[filePath]; !exists {
		bm.Bookmarks[filePath] = []BookmarkEntry{}
	}

	// Add bookmark
	bm.Bookmarks[filePath] = append(bm.Bookmarks[filePath], bookmark)
	// Sort bookmarks for this file
	bm.SortBookmarksForTrack(filePath)
	// Get the index of the newly added bookmark
	return bm.GetBookmarkIndex(filePath, bookmark)
}

func (bm *BookmarksManager) GetBookmarkIndex(filePath string, bookmark BookmarkEntry) int {
	bookmarks, exists := bm.Bookmarks[filePath]
	if !exists {
		return -1
	}
	for i, b := range bookmarks {
		if b == bookmark {
			return i
		}
	}
	return -1
}

func (bm *BookmarksManager) SortBookmarksForTrack(filepath string) {
	// Sort bookmarks for the given file path
	bookmarks := bm.Bookmarks[filepath]
	if len(bookmarks) > 0 {
		// Sort bookmarks in ascending order
		for i := 0; i < len(bookmarks)-1; i++ {
			for j := i + 1; j < len(bookmarks); j++ {
				if bookmarks[i] > bookmarks[j] {
					bookmarks[i], bookmarks[j] = bookmarks[j], bookmarks[i]
				}
			}
		}
	}
}

// RemoveBookmark removes a specific bookmark for a file
func (bm *BookmarksManager) RemoveBookmark(filePath string, bookmark BookmarkEntry) bool {
	bookmarks, exists := bm.Bookmarks[filePath]
	if !exists {
		return false
	}

	for i, b := range bookmarks {
		if b == bookmark {
			// Remove the bookmark by slicing it out
			bm.Bookmarks[filePath] = append(bookmarks[:i], bookmarks[i+1:]...)
			return true
		}
	}

	return false
}

// GetBookmarks returns all bookmarks for a specific file
func (bm *BookmarksManager) GetBookmarks(filePath string) []BookmarkEntry {
	return bm.Bookmarks[filePath]
}

// ClearBookmarks removes all bookmarks for a specific file
func (bm *BookmarksManager) ClearBookmarks(filePath string) {
	delete(bm.Bookmarks, filePath)
}

// Update tabs based on bookmarks
func (m *model) UpdateBookmarkTabs() {
	defaultTabs := []string{"No Bookmarks", "<b>"}

	// If no track is selected or playing, keep default tabs
	if currentTrackIndex < 0 || currentTrackIndex >= len(trackList) {
		m.Tabs = defaultTabs
		return
	}

	// Get bookmarks for current track
	bookmarks := bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename)

	// If no bookmarks, show default tabs
	if len(bookmarks) == 0 {
		m.Tabs = defaultTabs
		return
	}

	// Create tabs for each bookmark
	newTabs := make([]string, 0, len(bookmarks)+1)

	// Add a tab for each bookmark
	for _, bookmark := range bookmarks {
		bookmarkLabel := bookmark.String()
		newTabs = append(newTabs, bookmarkLabel)
	}

	// Add the "Add Bookmark" tab at the end
	newTabs = append(newTabs, "<b>")

	// Update model with new tabs
	m.Tabs = newTabs

	// Ensure active tab is within bounds
	if m.activeTab >= len(m.Tabs) {
		m.activeTab = len(m.Tabs) - 1
	}

	log.Printf("Updated tabs: %v", m.Tabs)
}

func JumpToBookmark(player *Player, bookmark BookmarkEntry) error {
	log.Printf("Attempting to jump to bookmark: %s", bookmark.String())
	// Parse the HH:MM:SS format to Seconds
	parts := strings.Split(bookmark.String(), ":")
	if len(parts) != 3 {
		log.Printf("Invalid bookmark format: %s", bookmark)
		return fmt.Errorf("invalid bookmark format: %s", bookmark)
	}

	hours, errH := strconv.Atoi(parts[0])
	minutes, errM := strconv.Atoi(parts[1])
	seconds, errS := strconv.Atoi(parts[2])

	if errH != nil || errM != nil || errS != nil {
		log.Printf("Invalid bookmark time format: %s", bookmark)
		return fmt.Errorf("invalid bookmark time format: %s", bookmark)
	}

	totalSeconds := hours*3600 + minutes*60 + seconds
	log.Printf("Bookmark position in seconds: %d", totalSeconds)

	// Convert seconds to a fraction of the total duration
	// Your SeekTo function expects a fraction between 0.0 and 1.0
	duration := player.total.Seconds()
	if duration <= 0 {
		log.Println("Invalid total duration")
		return fmt.Errorf("invalid total duration")
	}

	fraction := float64(totalSeconds) / duration

	// Use the player's SeekTo function to jump to the bookmark
	log.Printf("Jumping to bookmark: %s at fraction: %.4f", bookmark, fraction)
	return player.SeekTo(fraction)
}
