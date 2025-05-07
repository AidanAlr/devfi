package main

import (
	"io"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type (
	changeVolume int
	seekPosition int
	tickMsg      time.Time
)

type audioKeyMap struct {
	Play           key.Binding
	TogglePause    key.Binding
	ToggleRepeat   key.Binding
	SeekRight      key.Binding
	SeekLeft       key.Binding
	Delete         key.Binding
	MoveUp         key.Binding
	MoveDown       key.Binding
	VolumeUp       key.Binding
	VolumeDown     key.Binding
	AddBookmark    key.Binding
	RemoveBookmark key.Binding
	NextBookmark   key.Binding
	PrevBookmark   key.Binding
}

var (
	player            *Player
	trackList         []track
	playstatus        string
	progressStr       string
	currentTrackIndex int
	bookmarksManager  = NewBookmarksManager()
)

type model struct {
	dump                 io.Writer
	list                 list.Model
	err                  error
	keys                 *audioKeyMap
	audioContext         *audio.Context
	Tabs                 []string
	activeTab            int
	progress             progress.Model
	repeat               bool
	volumeChange         int
	seekChange           int
	initialWindowWidth   int
	currentBookmarkIndex int
}

func (m model) View() string {
	// First, create a top-level container
	var container strings.Builder

	// Render tabs section - now showing bookmarks
	var renderedTabs []string
	for i, t := range m.Tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(m.Tabs)-1, i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		} else if isLast && isActive {
			border.BottomRight = "│"
		} else if isLast && !isActive {
			border.BottomRight = "┤"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	container.WriteString(tabsRow)
	container.WriteString("\n")

	// Calculate width slightly smaller than terminal width
	terminalWidth := m.initialWindowWidth

	// Make container slightly smaller than terminal
	containerWidth := int(float64(terminalWidth) * 0.98)

	// Account for borders and padding
	contentWidth := containerWidth - 4 // Subtract border and padding width
	currentWidth, currentHeight := m.list.Width(), m.list.Height()

	// Apply the restricted dimensions
	m.list.SetSize(contentWidth, 20) // Height of 20 rows, width limited

	// Set up the progress bar with limited width
	originalProgressWidth := m.progress.Width

	// Set progress width to match the content width
	m.progress.Width = contentWidth - len(playstatus) - len(progressStr) - 2

	// Create the progress bar view
	progressBar := m.progress.View() + playstatus + progressStr

	// Get the list view with restricted dimensions
	listView := m.list.View()

	// Restore original dimensions
	m.list.SetSize(currentWidth, currentHeight)
	m.progress.Width = originalProgressWidth

	// 5. Create a unified player section with controlled width
	playerContent := progressBar + "\n\n" + listView

	// Apply consistent styling with calculated width
	playerStyle := lipgloss.NewStyle().
		Width(containerWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(highlightColor).
		Padding(1, 1)

	playerContainer := playerStyle.Render(playerContent)

	// Add the unified player container to our view
	container.WriteString(playerContainer)

	// Return the final view
	return docStyle.Render(container.String())
}

func newAudioKeyMap() *audioKeyMap {
	log.Println("Initializing audio key mapping")
	return &audioKeyMap{
		Play: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		TogglePause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle pause"),
		),
		ToggleRepeat: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "toggle repeat"),
		),
		SeekRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "seek right"),
		),
		SeekLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "seek left"),
		),
		VolumeUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "increase volume"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "decrease volume"),
		),
		AddBookmark: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "add bookmark"),
		),
		RemoveBookmark: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "remove bookmark"),
		),
		NextBookmark: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next bookmark"),
		),
		PrevBookmark: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "previous bookmark"),
		),
	}
}

func New(trackList []list.Item) *model {
	log.Println("Initializing application model")
	// Create our key map
	keyList := newAudioKeyMap()

	// Initialize empty tabs and tab content
	initialTabs := []string{}

	// Initialize the model with the key map and empty tabs
	newModel := model{
		keys:      newAudioKeyMap(),
		Tabs:      initialTabs,
		activeTab: 0,
	}

	// Create a custom delegate with our desired help items
	delegate := list.NewDefaultDelegate()

	// Set the delegate's help functions to show our preferred keys
	delegate.ShortHelpFunc = func() []key.Binding {
		return []key.Binding{
			// Play and pause
			keyList.Play,        // Enter key to play
			keyList.TogglePause, // Space key to toggle pause

			// Bookmark navigation
			keyList.AddBookmark,    // Add bookmark
			keyList.RemoveBookmark, // Delete bookmark
			keyList.NextBookmark,   // Next bookmark
			keyList.PrevBookmark,   // Previous bookmark

		}
	}

	delegate.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{
			{
				// Playback
				keyList.Play,
				keyList.TogglePause,
				keyList.ToggleRepeat,
				keyList.SeekLeft,
				keyList.SeekRight,
			},
			{
				// Volume
				keyList.VolumeUp,
				keyList.VolumeDown,

				// Bookmarks
				keyList.AddBookmark,
				keyList.RemoveBookmark,
				keyList.NextBookmark,
				keyList.PrevBookmark,
			},
			{
				// UI controls
				list.DefaultKeyMap().ShowFullHelp,
				list.DefaultKeyMap().Quit,
			},
		}
	}

	// Add the trackList with our custom delegate
	newModel.list = list.New(trackList, delegate, 0, 0)

	// Set the title
	newModel.list.Title = "TrackList"

	newModel.list.KeyMap.Filter.SetEnabled(true)
	newModel.list.KeyMap.CursorUp = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "up"),
	)
	newModel.list.KeyMap.CursorDown = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "down"),
	)
	newModel.list.KeyMap.NextPage.SetEnabled(false)
	newModel.list.KeyMap.PrevPage.SetEnabled(false)

	newModel.progress = progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	playstatus = "❚❚"
	newModel.audioContext = audio.NewContext(sampleRate)
	return &newModel
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	accentColor       = lipgloss.AdaptiveColor{Light: "#FF4D8B", Dark: "#FF5F9D"}

	// Inactive tab style
	inactiveTabStyle = lipgloss.NewStyle().
				Border(inactiveTabBorder, true).
				BorderForeground(highlightColor).
				Padding(0, 1)

	// Active tab with visual enhancement
	activeTabStyle = lipgloss.NewStyle().
			Border(activeTabBorder, true).
			BorderForeground(accentColor).
			Foreground(accentColor).
			Bold(true).   // Make text bold
			Padding(0, 3) // Add more horizontal padding

	windowStyle = lipgloss.NewStyle().
			BorderForeground(highlightColor).
			Padding(2, 0).
			Align(lipgloss.Center).
			Border(lipgloss.NormalBorder()).
			UnsetBorderTop()
)

func (m model) Init() tea.Cmd {
	log.Println("Initializing tea model")
	return m.TickUpdate(0.0)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Update function is repeatedly called asynchronously and is responsible for rendering to the view
	// Get the input msg
	switch msg := msg.(type) {
	// WindowSizeMsg is used to resize the window
	case tea.WindowSizeMsg:
		log.Printf("Window resized to %d x %d", msg.Width, msg.Height)
		m.initialWindowWidth = msg.Width - 2 - len(playstatus)
		m.progress.Width = m.initialWindowWidth

		// IMPORTANT: Reserve space for tabs and progress bar
		// Estimate how much space tabs and progress will take
		tabsHeight := 4     // Rough estimate for tabs section
		progressHeight := 2 // Rough estimate for progress section

		// Calculate remaining height for the list
		listHeight := msg.Height - tabsHeight - progressHeight - 2 // -2 for padding

		// Set the list size with the calculated height
		m.list.SetSize(msg.Width, listHeight)
		return m, nil

	// Handle tick messages for updating the progress bar
	case tickMsg:
		// Default progress to 0
		fracDone := 0.0
		// If a player is active
		if player != nil {
			// Update the progress string and get fraction of completion
			progressStr, fracDone = player.Update()
			// Adjust progress bar width based on the length of the progress text
			m.progress.Width = m.initialWindowWidth - len(progressStr)
		}
		// Update the progress bar with the new percentage
		cmd := m.progress.SetPercent(fracDone)
		// Return a batch of commands: continue ticking and update progress
		return m, tea.Batch(m.TickUpdate(fracDone), cmd)

	// Handle progress bar frame updates
	case progress.FrameMsg:
		// Update the progress bar model
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	// Handle volume change messages
	case changeVolume:
		// Check if this is the most recent volume change request
		if int(msg) == m.volumeChange {
			log.Printf("Applying volume change: %d", m.volumeChange)
			// Apply the volume change
			player.SetVolume(m.volumeChange)
			// Reset the volume change counter
			m.volumeChange = 0
			return m, nil
		}

	// Handle seek position messages
	case seekPosition:
		// Check if this is the most recent seek request
		if int(msg) == m.seekChange {
			log.Printf("Applying seek position: %d", m.seekChange)
			// Apply the seek operation
			player.Seek(m.seekChange)
			// Reset the seek change counter
			m.seekChange = 0
			return m, nil
		}
	// KeyMsg is used to handle key events
	case tea.KeyMsg:
		// Check if the list is in filtering mode
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		// Handle bookmark actions
		case key.Matches(msg, m.keys.AddBookmark):
			{
				if player == nil {
					// We cant add a bookmark if there is no player
					break
				}
				log.Println("Adding bookmark")
				// Get the current position of the player in the track
				current := player.audioPlayer.Current()
				// Convert the current position to a bookmark string
				bookmark := bookmarksManager.ConvertDurationToBookmark(current)
				// Add the bookmark to the bookmarks manager
				index := bookmarksManager.AddBookmark(trackList[currentTrackIndex].filename, bookmark)
				log.Printf("Added bookmark at position: %s for track: %s", bookmark, trackList[currentTrackIndex].filename)
				// Update the current bookmark index
				m.currentBookmarkIndex = index
				// Update tabs to reflect the new bookmarks
				m.UpdateBookmarkTabs()
				// Set the active tab to the new bookmark
				m.activeTab = index
			}
		case key.Matches(msg, m.keys.RemoveBookmark):
			{
				if player == nil {
					// We cant remove a bookmark if there is no player
					break
				}
				log.Println("Removing bookmark")
				// Get the currently selected bookmark
				bookmarks := bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename)
				// Valid bookmark to delete
				if len(bookmarks) > 0 && m.currentBookmarkIndex < len(bookmarks) {
					bookmark := bookmarks[m.currentBookmarkIndex]
					log.Printf("Removing bookmark: %s from track: %s", bookmark.String(), trackList[currentTrackIndex].filename)
					// Delete the currentBookmarkIndex
					bookmarksManager.RemoveBookmark(trackList[currentTrackIndex].filename, bookmark)
					// Update tabs
					m.UpdateBookmarkTabs()
					// Adjust current index if needed
					if m.currentBookmarkIndex >= len(bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename)) {
						m.currentBookmarkIndex = max(0, len(bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename))-1)
					}
					m.activeTab = min(m.currentBookmarkIndex, len(m.Tabs)-2)
				}
			}
		case key.Matches(msg, m.keys.NextBookmark):
			{
				log.Println("Moving to next bookmark")
				bookmarks := bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename)
				if len(bookmarks) > 0 {
					// Move to next bookmark, wrapping around if necessary
					m.currentBookmarkIndex = (m.currentBookmarkIndex + 1) % len(bookmarks)
					// Set the active tab to match
					m.activeTab = m.currentBookmarkIndex
					// Get the bookmark at the current index
					bookmark := bookmarks[m.currentBookmarkIndex]
					log.Printf("Moving to bookmark: %s", bookmark.String())
					JumpToBookmark(player, bookmark)
				} else {
					log.Printf("No bookmarks available for track: %s", trackList[currentTrackIndex].filename)
				}
			}
		case key.Matches(msg, m.keys.PrevBookmark):
			{
				log.Println("Moving to previous bookmark")
				bookmarks := bookmarksManager.GetBookmarks(trackList[currentTrackIndex].filename)
				if len(bookmarks) > 0 {
					// Move to previous bookmark, wrapping around if necessary
					if m.currentBookmarkIndex <= 0 {
						m.currentBookmarkIndex = len(bookmarks) - 1
					} else {
						m.currentBookmarkIndex--
					}
					// Set the active tab to match
					m.activeTab = m.currentBookmarkIndex
					// Get the bookmark at the current index
					bookmark := bookmarks[m.currentBookmarkIndex]
					log.Printf("Moving to bookmark: %s", bookmark.String())
					JumpToBookmark(player, bookmark)
				} else {
					log.Printf("No bookmarks available for track: %s", trackList[currentTrackIndex].filename)
				}
			}

		// Check if the user pressed the quit key
		case key.Matches(msg, list.DefaultKeyMap().Quit):
			log.Println("Quit requested")
			if player != nil {
				// Close the audio player
				player.Close()
				log.Println("Closed audio player")
			}

			// Save the bookmarks added or removed in this session
			bookmarksManager.SaveBookmarks()
			log.Println("Saved bookmarks and exiting application")
			// Quit the program
			return m, tea.Quit
		// Check if the user pressed the select and play key
		case key.Matches(msg, m.keys.Play):
			// Get the currently selected track
			currentTrackIndex = m.list.Index()
			log.Printf("Selected track index: %d", currentTrackIndex)
			// If a player already exists, i.e. another track is playing
			if player != nil {
				// Restart the track
				log.Println("Rewinding current track")
				player.Rewind(currentTrackIndex)
			}
			// Create a new player for the selected track
			selectedTrack := trackList[currentTrackIndex]
			log.Printf("Playing track: %s", selectedTrack.title)
			var err error
			player, err = NewPlayer(selectedTrack.file, m.audioContext, selectedTrack.audioformat)
			if err != nil {
				// We received an error attempting to play the track
				log.Printf("Error playing track: %v", err)
				m.list.NewStatusMessage("Cannot Play Audio: " + err.Error())
				progressStr = "--/--"
				m.progress.Width = m.initialWindowWidth - len(progressStr)
			} else {
				// Show the pause button, to indicate that the track is playing
				playstatus = " ❚❚ "
				// Update the title of the list to show the currently playing track
				m.list.Title = m.list.SelectedItem().(track).title
				log.Printf("Now playing: %s", m.list.Title)

				// Update bookmark tabs for the new track
				m.UpdateBookmarkTabs()
			}
			return m, nil

		// Check if the user pressed the volume up key
		case key.Matches(msg, m.keys.VolumeUp):
			// If a player is active
			if player != nil {
				// Increment volume change counter
				m.volumeChange++
				log.Printf("Volume up requested, new value: %d", m.volumeChange)
				// This batches rapid key presses together
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return changeVolume(m.volumeChange)
				})
			}
			return m, nil

		// Check if the user pressed the volume down key
		case key.Matches(msg, m.keys.VolumeDown):
			// If a player is active
			if player != nil {
				// Decrement volume change counter
				m.volumeChange--
				log.Printf("Volume down requested, new value: %d", m.volumeChange)
				// Schedule a volume change event after a short delay
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return changeVolume(m.volumeChange)
				})
			}
			return m, nil

		// Check if the user pressed the seek right key
		case key.Matches(msg, m.keys.SeekRight):
			// If a player is active
			if player != nil {
				// Increment seek position counter
				m.seekChange++
				log.Printf("Seek forward requested, value: %d", m.seekChange)
				// Schedule a seek position event after a short delay
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return seekPosition(m.seekChange)
				})
			}
			return m, nil

		// Check if the user pressed the seek left key
		case key.Matches(msg, m.keys.SeekLeft):
			// If a player is active
			if player != nil {
				// Decrement seek position counter
				m.seekChange--
				log.Printf("Seek backward requested, value: %d", m.seekChange)
				// Schedule a seek position event after a short delay
				return m, tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
					return seekPosition(m.seekChange)
				})
			}
			return m, nil

		// Check if the user pressed the toggle pause key
		case key.Matches(msg, m.keys.TogglePause):
			// If a player is active
			if player != nil {
				log.Println("Toggling pause state")
				// Toggle pause state and update the play status indicator
				playstatus = player.TogglePause()
			}
			return m, nil

		// Check if the user pressed the toggle repeat key
		case key.Matches(msg, m.keys.ToggleRepeat):
			// Toggle repeat mode flag
			m.repeat = !m.repeat
			log.Printf("Toggling repeat mode: %v", m.repeat)
			return m, nil
		}

	}
	// If we reach here, update the list model and return any commands
	// We will reach here for filtering
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// TickUpdate handles ticking for the progress bar and transitions to the next song when complete
func (m *model) TickUpdate(fracDone float64) tea.Cmd {
	// If the track has finished playing (progress = 100%)
	if fracDone == 1.0 {
		log.Println("Track finished playing, moving to next song")
		// Play the next song based on repeat settings
		m.NextTrack()
	}
	// Continue the tick loop every 100ms
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
