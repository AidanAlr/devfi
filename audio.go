package main

import (
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// Audio configuration constants
const (
	sampleRate     = 44100 // Sample rate in Hz (CD quality is 44100)
	bytesPerSample = 4     // Number of bytes per sample (stereo = 4 bytes)
)

// audioType defines the different audio formats supported by the player
type audioType int

const (
	typeMP3 audioType = iota // MP3 audio format
)

// Player represents the audio player with playback control capabilities
type Player struct {
	audioContext *audio.Context // The global audio context from Ebiten
	audioPlayer  *audio.Player  // The actual player instance handling the audio
	totalStr     string         // Human-readable string of total track duration
	total        time.Duration  // Total track duration as a time.Duration value
	volume       int            // Current volume
	audioType    audioType      // Format of the audio being played
}

// NewPlayer creates a new player for the provided audio file
func NewPlayer(file *os.File, audioContext *audio.Context, audioType audioType) (*Player, error) {
	// Decode the audio file
	if audioType != typeMP3 {
		// We only handle MP3 files
		panic(" Unsupported audio type")
	}
	var err error

	// Decode MP3 file to stream with the specified sample rate
	s, err := mp3.DecodeWithSampleRate(sampleRate, file)
	if err != nil {
		return nil, err
	}

	// Create a new player from the audio stream
	p, err := audioContext.NewPlayer(s)
	if err != nil {
		return nil, err
	}

	// Initialize the Player struct with the newly created player
	player := &Player{
		audioContext: audioContext,
		audioPlayer:  p,
		// Calculate the total duration based on audio stream length
		// Formula: (bytes in stream) / (bytes per sample) / (samples per second) = seconds
		total:     time.Second * time.Duration(s.Length()) / bytesPerSample / sampleRate,
		volume:    85, // Set default volume
		audioType: audioType,
	}

	// Ensure we never have a zero duration (would cause division by zero later)
	if player.total == 0 {
		player.total = 1
	}

	// Store the string representation of the total duration
	player.totalStr = player.total.Truncate(time.Second).String()

	// Start playing the audio
	player.audioPlayer.Play()

	return player, nil
}

// Close releases resources associated with the player
func (p *Player) Close() error {
	return p.audioPlayer.Close()
}

// TogglePause toggles the playback state between playing and paused
// Returns a string representing the current state (for UI display)
func (p *Player) TogglePause() string {
	if p.audioPlayer.IsPlaying() {
		p.audioPlayer.Pause()
		return " ▶ " // Switch to Play icon
	}
	p.audioPlayer.Play()
	return " ❚❚ " // Switch to Pause icon
}

// Rewind restarts the track at the beginning
// Takes the track index to handle file seeking if needed
func (p *Player) Rewind(idx int) {
	if p.audioPlayer.IsPlaying() {
		// For playing tracks, use the player's rewind functionality
		p.audioPlayer.Rewind()
		p.Close()
	} else {
		// For paused tracks, seek the file to the beginning
		trackList[idx].file.Seek(0, 0)
	}
}

// SetVolume adjusts the player volume based on the provided change
func (p *Player) SetVolume(change int) {
	// Clamp volume to valid range
	if p.volume+change > 100 {
		p.volume = 100
	} else if p.volume+change < 0 {
		p.volume = 0
	} else {
		p.volume += change
	}
	// Apply normalized volume to the player (0.0-1.0 range)
	p.audioPlayer.SetVolume(float64(p.volume) / 100)
}

// SeekTo jumps to a specific position in the track
// Fraction of the total duration (0.0-1.0)
func (p *Player) SeekTo(fraction float64) error {
	// Calculate absolute position in milliseconds
	position := time.Duration(float64(p.total.Milliseconds()) * fraction * float64(time.Millisecond))

	// Clamp position to track boundaries
	if position > p.total {
		position = p.total
	}
	if position < 0 {
		position = 0
	}

	// Perform the seek operation
	if err := p.audioPlayer.Seek(position); err != nil {
		return err
	}
	return nil
}

// Seek advances or rewinds the playback position by the specified number of seconds
func (p *Player) Seek(change int) error {
	// Calculate new position by adding the change (in seconds) to current position
	position := p.audioPlayer.Current() + time.Second*time.Duration(change)

	// Clamp position to track boundaries
	if position > p.total {
		position = p.total
	}
	if position < 0 {
		position = 0
	}

	// Perform the seek operation
	if err := p.audioPlayer.Seek(position); err != nil {
		return err
	}
	return nil
}

// Update returns the current playback status
// Returns a formatted time string and completion fraction (0.0-1.0)
func (p *Player) Update() (string, float64) {
	// Get current playback position
	current := p.audioPlayer.Current()

	// Calculate fraction of track completed
	timeFrac := float64(current.Milliseconds()) / float64(p.total.Milliseconds())

	// Return formatted string "current / total" and completion fraction
	// This is used for the progress bar in the UI
	return current.Truncate(time.Second).String() + " / " + p.totalStr, timeFrac
}

// NextTrack attempts to play the next track in the tracklist
// If repeat mode is enabled, it will replay the current track
// If repeat mode is disabled, it will advance to the next track
// The function will continue trying tracks until it finds one that can be played successfully
func (m *model) NextTrack() {
	log.Printf("Moving to next track, repeat mode: %v", m.repeat)
	// If a player exists, rewind the current track
	if player != nil {
		player.Rewind(currentTrackIndex)
	}

	var err error
	// Get the total number of tracks in the list
	trackCount := len(trackList)

	// Keep trying tracks until we find one that works
	for {
		// If not in repeat mode, move to the next track
		if !m.repeat {
			// Increment the current index
			currentTrackIndex++
			// Wrap around to the beginning if we've reached the end
			currentTrackIndex %= trackCount
			log.Printf("Moving to track index: %d", currentTrackIndex)
		} else {
			log.Println("Repeat mode active, replaying current track")
		}
		// Otherwise, in repeat mode, we'll try the same track again

		// Get the selected track
		selectedTrack := trackList[currentTrackIndex]
		log.Printf("Attempting to play track: %s", selectedTrack.title)
		// Attempt to create a new player for the selected track
		player, err = NewPlayer(selectedTrack.file, m.audioContext, selectedTrack.audioformat)

		// If player creation was successful
		if err == nil {
			// Update the selected item in the UI list
			m.list.Select(currentTrackIndex)
			// Set play status indicator
			playstatus = " ❚❚ "
			// Update the list title to show the current track
			m.list.Title = m.list.SelectedItem().(track).title
			log.Printf("Successfully playing track: %s", m.list.Title)

			// Update bookmark tabs for the new track
			m.UpdateBookmarkTabs()
			// Exit the loop as we've found a playable track
			break
		}

		// If we couldn't play the track, show an error message
		log.Printf("Error playing track: %v", err)
		m.list.NewStatusMessage("Cannot Play Audio: " + err.Error())
		// Update progress display to show error state
		progressStr = "--/--"
		// Adjust progress bar width
		m.progress.Width = m.initialWindowWidth - len(progressStr)

		// If in repeat mode and the track fails to play, don't try other tracks
		if m.repeat {
			log.Println("In repeat mode with unplayable track, stopping attempts")
			break
		}
		// If not in repeat mode, the loop will continue and try the next track
	}
}
