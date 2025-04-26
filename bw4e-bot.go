package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
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

var config Config
var sheetsService *sheets.Service

func main() {
	log.Println("Loading configuration...")
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer configFile.Close()

	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		log.Fatalf("Error decoding config file: %v", err)
	}

	log.Println("Initializing Google Sheets API...")
	authJSON, err := os.ReadFile(config.CredentialsPath)
	if err != nil {
		log.Fatalf("Error reading credentials file: %v", err)
	}

	configGoogle, err := google.JWTConfigFromJSON(authJSON, sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		log.Fatalf("Error initializing Google Sheets API: %v", err)
	}
	sheetsService, err = sheets.New(configGoogle.Client(nil))
	if err != nil {
		log.Fatalf("Error creating Sheets service: %v", err)
	}

	log.Println("Creating Discord session...")
	dg, err := discordgo.New("Bot " + config.BotToken)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers

	log.Println("Adding event handlers...")
	dg.AddHandler(onMessageCreate)
	dg.AddHandler(onInteractionCreate)

	log.Println("Connecting to Discord...")
	if err := dg.Open(); err != nil {
		log.Fatalf("Error opening WebSocket connection: %v", err)
	}
	defer dg.Close()

	log.Println("Registering commands...")
	registerCommands(dg)

	fmt.Println("Bot is now running. Press Ctrl+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}

// Register guild-specific slash commands
func registerCommands(s *discordgo.Session) {
	if s == nil || s.State == nil || s.State.User == nil {
		log.Fatalf("Cannot register commands: Discord session state is not initialized.")
	}

	// No commands to register after removing /hide and /unhide
}

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

// Handle interactions for slash commands
func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Intentionally left blank: ignore all slash commands
}
