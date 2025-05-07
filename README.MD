# Terminal MP3 Player With Bookmark Management

This project is a keyboard driven terminal-based MP3 player written in Go, designed for podcast and audiobook enthusiasts. Its standout feature is persistent bookmarking functionality, allowing you to save and resume playback positions across sessions.
The user interface is powered by [BubbleTea](https://github.com/charmbracelet/bubbletea)

## Key Features

- **Track Management**: Browse, filter, and play MP3 files from multiple directories
- **Bookmark System**: Add, remove, and navigate between bookmarks within each track
- **Playback Controls**: Play/pause, seek, adjust volume, and toggle repeat mode
- **Persistent Settings**: Bookmarks are saved between sessions
- **Intuitive UI**: Clean terminal interface with progress bar and visual bookmark tabs

## Configuration

On first run, you'll be prompted to create a configuration file specifying which directories to scan for MP3 files. Create the file with this command (modify the paths to your own music/podcast directories):

```bash
echo '{
  "directories": [
    "/Users/yourusername/Music",
    "/Users/yourusername/Podcasts"
  ]
}' > ~/.audioplayer-config.json
```

## Bookmark System

- Bookmarks are automatically stored in `~/.bookmarks` (file is created if it doesn't exist)
- Each audio file can have multiple bookmarks
- Bookmarks appear as tabs in the UI for easy navigation
- Bookmarks are displayed in chronological order

## Keyboard Controls

| Key           | Action                 | Description                                     |
|---------------|------------------------|-------------------------------------------------|
| Enter         | Play                   | Select and play the highlighted track           |
| Space         | Toggle Pause           | Pause/resume the currently playing track        |
| r             | Toggle Repeat          | Toggle repeat mode on/off                       |
| ←             | Seek Left              | Seek backward in the current track              |
| →             | Seek Right             | Seek forward in the current track               |
| ↑             | Volume Up              | Increase playback volume                        |
| ↓             | Volume Down            | Decrease playback volume                        |
| b             | Add Bookmark           | Add a bookmark at the current position          |
| B             | Remove Bookmark        | Remove the current bookmark                     |
| n             | Next Bookmark          | Jump to the next bookmark                       |
| p             | Previous Bookmark      | Jump to the previous bookmark                   |
| k             | Up                     | Move up in the track list                       |
| j             | Down                   | Move down in the track list                     |
| /             | Filter                 | Filter tracks by title                          |
| q             | Quit                   | Exit the application                            |

## Project Structure

```
audioplayer
├── audio.go          # Audio playback core functionality
├── bookmarks.go      # Bookmark management system
├── config.go         # Configuration file handling
├── main.go           # Application entry point
├── track.go          # Track data structure and file scanning
├── ui.go             # User interface components and event handling
└── utils.go          # Helper functions and utilities
```

### audio.go (Audio Playback)

#### Types & Structs:
- `musicType`: Enum for audio formats (currently only supports MP3)
- `Player`: Core struct handling audio playback with fields for:
  - `audioContext`: Manages audio context from Ebiten
  - `audioPlayer`: Handles actual audio playback
  - `totalStr`: Human-readable duration string
  - `total`: Duration as time.Duration value
  - `volume`: Current volume level (0-128)
  - `musicType`: Format of the audio being played

#### Key Functions:
- `NewPlayer`: Creates a player for an audio file, setting up decoding and playback
- `Close`: Releases player resources
- `TogglePause`: Toggles between play/pause, returns UI icon
- `Rewind`: Restarts track from beginning
- `SetVolume`: Adjusts volume level with range checking
- `SeekTo`: Jumps to position based on track fraction (0.0-1.0)
- `Seek`: Moves position by specific seconds
- `Update`: Returns current playback status for UI updates

### bookmarks.go (Bookmark Management)

#### Types & Structs:
- `BookmarkEntry`: String representation of a timestamp in "HH:MM:SS" format
- `BookmarkMap`: Maps filepaths to bookmark lists
- `BookmarksManager`: Handles bookmark persistence

#### Key Functions:
- `NewBookmarksManager`: Creates manager, initializes file storage
- `ConvertDurationToBookmark`: Formats time.Duration as "HH:MM:SS"
- `LoadBookmarks`: Loads bookmarks from JSON file
- `SaveBookmarks`: Persists bookmarks to JSON file
- `AddBookmark`: Adds bookmark for a file, returns its index
- `RemoveBookmark`: Removes a specific bookmark
- `GetBookmarks`: Retrieves all bookmarks for a file
- `GetBookmarkIndex`: Finds position of bookmark in list
- `SortBookmarksForTrack`: Sorts bookmarks chronologically
- `ClearBookmarks`: Removes all bookmarks for a file
- `JumpToBookmark`: Navigates to bookmark position in track

### config.go (Configuration)

#### Types & Structs:
- `Config`: Contains directories to search for MP3 files

#### Key Functions:
- `GetUserHomeDir`: Gets user's home directory path
- `FindConfigFile`: Locates config file, prompts creation if missing
- `LoadConfig`: Loads configuration from disk

### track.go (Track Management)

#### Types & Structs:
- `track`: Represents a playable audio track with metadata

#### Key Functions:
- `UpdateTrackList`: Recursively scans directories for MP3 files
  - Identifies MP3 files by extension
  - Extracts metadata and creates track objects
  - Populates global tracklist for playback

### ui.go (User Interface)

#### Types & Structs:
- `model`: Main application model for UI and state management
- `audioKeyMap`: Maps keyboard controls to player functions

#### Key Functions:
- `New`: Creates application model with keymap and UI components
- `Update`: Main event loop handling keyboard, progress, and ticks
- `View`: Renders the complete UI with tabs, player, and tracklist
- `TickUpd`: Handles progress updates and track transitions
- `NextSong`: Handles track transitions and repeat mode

### utils.go (Utilities)

#### Key Functions:
- `TitleDescriptionAudioFormat`: Extracts track metadata
  - Reads ID3 tags from MP3 files
  - Handles missing metadata gracefully with defaults
- Helper functions: `min` and `max`

### main.go (Application Entry)

- Initializes application components
- Sets up logging
- Loads configuration
- Starts the terminal UI
