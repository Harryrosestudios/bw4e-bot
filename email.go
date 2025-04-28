package main

import (
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Fetch emails from Google Sheets
func fetchEmailsFromSheet() []string {
	log.Println("Fetching emails from Google Sheets...")
	resp, err := sheetsService.Spreadsheets.Values.Get(config.SpreadsheetID, "Sheet1!A:A").Do()
	if err != nil {
		log.Printf("Error fetching emails from Google Sheets: %v", err)
		return []string{}
	}

	var emails []string
	for _, row := range resp.Values {
		if len(row) > 0 {
			emails = append(emails, strings.ToLower(row[0].(string)))
		}
	}
	log.Printf("Fetched emails: %v", emails)
	return emails
}

// Handle message creation events (email verification)
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("Received message in channel %s from user %s: %s\n", m.ChannelID, m.Author.Username, m.Content)

	// Ignore messages from the bot itself or messages not in the specified channel
	if m.Author.ID == s.State.User.ID || m.ChannelID != config.EmailChannelID {
		log.Println("Ignoring message (either from bot or wrong channel).")
		return
	}

	// Validate email format
	emailRegex := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	isValidEmail := regexp.MustCompile(emailRegex).MatchString(m.Content)
	if !isValidEmail {
		log.Println("Invalid email format detected.")
		// Attempt to delete invalid message
		err := s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			log.Printf("Error deleting invalid email message: %v", err)
		}

		// Attempt to DM the user about invalid email
		dmChannel, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			log.Printf("Error creating DM channel: %v", err)
			return
		}
		_, err = s.ChannelMessageSend(dmChannel.ID, "Invalid email format. Please try again.")
		if err != nil {
			log.Printf("Error sending DM to user: %v", err)
		}
		return
	}

	log.Println("Valid email detected:", m.Content)

	email := strings.ToLower(strings.TrimSpace(m.Content))
	emails := fetchEmailsFromSheet()

	roleID := config.RoleNotFoundID
	message := "Your email wasn't found. Please create a support ticket."

	for _, e := range emails {
		if e == email {
			roleID = config.RoleFoundID
			message = ""
			break
		}
	}

	member, err := s.GuildMember(config.GuildID, m.Author.ID)
	if err != nil {
		log.Printf("Error fetching guild member: %v", err)
		return
	}

	log.Printf("Assigning role %s to user %s\n", roleID, m.Author.Username)
	if addErr := s.GuildMemberRoleAdd(config.GuildID, member.User.ID, roleID); addErr != nil {
		log.Printf("Error assigning role: %v", addErr)
		return
	}

	// Send a DM to the user if their email wasn't found
	if message != "" {
		dmChannel, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			log.Printf("Error creating DM channel: %v", err)
			return
		}
		_, err = s.ChannelMessageSend(dmChannel.ID, message)
		if err != nil {
			log.Printf("Error sending DM to user: %v", err)
		}
	}

	// Delete the original message after processing
	log.Println("Deleting original message...")
	err = s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		log.Printf("Error deleting original message: %v", err)
	}
}
