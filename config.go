package main

import (
	"encoding/json"
	"os"
)

// Config structure to hold configuration data
type Config struct {
	BotToken        string `json:"discordToken"`
	GuildID         string `json:"guildId"`
	EmailChannelID  string `json:"emailChannelId"`
	RoleFoundID     string `json:"roleFoundId"`
	RoleNotFoundID  string `json:"roleNotFoundId"`
	CredentialsPath string `json:"credentialsPath"`
	SpreadsheetID   string `json:"spreadsheetId"`
}

// Global config variable
var config Config

// Helper to load config from a file
func loadConfig(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&config)
}
