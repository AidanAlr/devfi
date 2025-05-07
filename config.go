package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
)

// Config contains a list of directories that the application will use to search for songs or podcasts
type Config struct {
	Directories []string `json:"directories"` // List of directories, serialized as "directories" in JSON
}

// getUserHomeDir gets the user's home directory path
func getUserHomeDir() string {
	// Get the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't determine the home directory, log the error and terminate
		log.Fatal("Cannot find Home Directory" + err.Error())
	}
	return home
}

// FindConfigFile searches for the config file in the user's home directory
// Returns the file content as a byte slice if found, nil otherwise
func FindConfigFile() []byte {
	// Build the path to the config file in the user's home directory
	configPath := path.Join(getUserHomeDir(), ".audioplayer-config.json")

	// Check if the file exists
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		// File exists, read its contents
		content, err := os.ReadFile(configPath)
		if err != nil {
			// If we can't read the file, log the error and terminate
			log.Fatal("Failed to read Config file" + err.Error())
		}
		return content
	}

	// If the file doesn't exist, print to the user asking them to create it
	// Example structure
	configData := map[string]interface{}{
		"directories": []string{
			"/Users/yourusername/directory",
		},
	}

	// Marshal with indentation
	jsonBytes, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		fmt.Println("Error creating JSON:", err)
	}

	// Convert to string and print instructions
	jsonString := string(jsonBytes)

	fmt.Println("Please create the config file ~/.audioplayer-config.json with the following structure:")
	fmt.Println()
	fmt.Println(jsonString)
	fmt.Println()
	fmt.Println("These directories should contain your mp3 files.")
	return nil
}

// LoadConfig loads the configuration from disk
// Returns a pointer to a Config struct if successful
// Exits the program with code -1 if the config file cannot be found or parsed
func LoadConfig() *Config {
	// Try to find and read the config file
	content := FindConfigFile()
	if content == nil {
		// Exit the program if config file can't be found
		os.Exit(-1)
	}

	// Initialize a new Config struct
	conf := new(Config)

	// Parse the JSON content into the Config struct
	err := json.Unmarshal(content, conf)
	if err != nil {
		// Exit the program if JSON parsing fails
		os.Exit(-1)
	}

	return conf
}
