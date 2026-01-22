package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerCredentials represents credentials for a single server
type ServerCredentials struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// CredentialsFile represents the structure of ~/.runqy/credentials.json
type CredentialsFile struct {
	Servers map[string]*ServerCredentials `json:"servers"`
	Current string                        `json:"current"`
}

// GetCredentialsDir returns the path to the credentials directory (~/.runqy/)
func GetCredentialsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".runqy"), nil
}

// GetCredentialsPath returns the path to the credentials file (~/.runqy/credentials.json)
func GetCredentialsPath() (string, error) {
	dir, err := GetCredentialsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// EnsureCredentialsDir creates the credentials directory with proper permissions
func EnsureCredentialsDir() error {
	dir, err := GetCredentialsDir()
	if err != nil {
		return err
	}

	// Create directory with user-only permissions (0700)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	return nil
}

// loadCredentialsFile reads and parses the credentials file
func loadCredentialsFile() (*CredentialsFile, error) {
	path, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty credentials if file doesn't exist
			return &CredentialsFile{
				Servers: make(map[string]*ServerCredentials),
				Current: "",
			}, nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var creds CredentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if creds.Servers == nil {
		creds.Servers = make(map[string]*ServerCredentials)
	}

	return &creds, nil
}

// saveCredentialsFile writes the credentials file with proper permissions
func saveCredentialsFile(creds *CredentialsFile) error {
	if err := EnsureCredentialsDir(); err != nil {
		return err
	}

	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with user-only permissions (0600)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// SaveCredentials saves credentials for a named server profile
func SaveCredentials(name, url, apiKey string) error {
	creds, err := loadCredentialsFile()
	if err != nil {
		return err
	}

	creds.Servers[name] = &ServerCredentials{
		URL:    url,
		APIKey: apiKey,
	}

	// Set as current if this is the first server or if it's "default"
	if creds.Current == "" || name == "default" {
		creds.Current = name
	}

	return saveCredentialsFile(creds)
}

// LoadCredentials loads credentials for a specific profile
func LoadCredentials(name string) (*ServerCredentials, error) {
	creds, err := loadCredentialsFile()
	if err != nil {
		return nil, err
	}

	server, ok := creds.Servers[name]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	return server, nil
}

// GetCurrentCredentials returns the active server profile's credentials
func GetCurrentCredentials() (*ServerCredentials, error) {
	creds, err := loadCredentialsFile()
	if err != nil {
		return nil, err
	}

	if creds.Current == "" {
		return nil, nil // No current profile set
	}

	server, ok := creds.Servers[creds.Current]
	if !ok {
		return nil, nil // Current profile doesn't exist
	}

	return server, nil
}

// GetCurrentProfileName returns the name of the current profile
func GetCurrentProfileName() (string, error) {
	creds, err := loadCredentialsFile()
	if err != nil {
		return "", err
	}
	return creds.Current, nil
}

// SetCurrentProfile switches the active profile
func SetCurrentProfile(name string) error {
	creds, err := loadCredentialsFile()
	if err != nil {
		return err
	}

	if _, ok := creds.Servers[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	creds.Current = name
	return saveCredentialsFile(creds)
}

// DeleteCredentials removes a profile
func DeleteCredentials(name string) error {
	creds, err := loadCredentialsFile()
	if err != nil {
		return err
	}

	if _, ok := creds.Servers[name]; !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(creds.Servers, name)

	// If we deleted the current profile, clear it or set to another
	if creds.Current == name {
		creds.Current = ""
		// Pick another profile if available
		for profileName := range creds.Servers {
			creds.Current = profileName
			break
		}
	}

	return saveCredentialsFile(creds)
}

// DeleteAllCredentials removes all saved credentials
func DeleteAllCredentials() error {
	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credentials file: %w", err)
	}

	return nil
}

// ListCredentials returns all saved profiles
func ListCredentials() (map[string]*ServerCredentials, string, error) {
	creds, err := loadCredentialsFile()
	if err != nil {
		return nil, "", err
	}

	return creds.Servers, creds.Current, nil
}

// MaskAPIKey returns a masked version of the API key for display
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}
