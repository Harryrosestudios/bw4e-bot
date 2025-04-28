package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// Global config and services
var sheetsService *sheets.Service

func main() {
	log.Println("Loading configuration...")
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("Error loading config file: %v", err)
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

	// Load partners from file
	if err := loadPartners(); err != nil {
		log.Fatalf("Error loading partners: %v", err)
	}
	// Load notification channels from file
	if err := loadNotificationChannels(); err != nil {
		log.Fatalf("Error loading notification channels: %v", err)
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

	// --- Only send embeds if not already present ---
	botUser, err := dg.User("@me")
	if err != nil {
		log.Fatalf("Could not get bot user: %v", err)
	}
	found, err := hasBotEmbedInChannel(dg, partnersChannelID, botUser.ID)
	if err != nil {
		log.Printf("Error checking for existing embed: %v", err)
	}
	if !found {
		sendPartnersEmbed(dg)
	}
	foundNotif, err := hasBotEmbedInChannel(dg, notificationsChannelID, botUser.ID)
	if err != nil {
		log.Printf("Error checking for existing notifications embed: %v", err)
	}
	if !foundNotif {
		sendNotificationsEmbed(dg)
	}
	// ------------------------------------------------

	fmt.Println("Bot is now running. Press Ctrl+C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	fmt.Println("Shutting down...")
}
