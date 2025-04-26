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

// --- Partner Feature Section ---

type Partner struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Offering    string `json:"offering"`
	LogoURL     string `json:"logo_url"`
	Link        string `json:"link"`
	Emoji       string `json:"emoji"`
}

var partners []Partner
const partnersFile = "partners.json"

const (
	partnersChannelID   = "1365774652986626109"
	accessRoleID        = "1311141977558876201"
	addPartnerRoleID    = "1311142224716890145"
)

func loadPartners() error {
	b, err := os.ReadFile(partnersFile)
	if err != nil {
		if os.IsNotExist(err) {
			partners = []Partner{}
			return nil
		}
		return err
	}
	return json.Unmarshal(b, &partners)
}

func savePartners() error {
	b, err := json.MarshalIndent(partners, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(partnersFile, b, 0644)
}

// NEW: Check for existing bot embed in the channel
func hasBotEmbedInChannel(s *discordgo.Session, channelID, botUserID string) (bool, error) {
	messages, err := s.ChannelMessages(channelID, 50, "", "", "")
	if err != nil {
		return false, err
	}
	for _, msg := range messages {
		if msg.Author != nil && msg.Author.ID == botUserID && len(msg.Embeds) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func sendPartnersEmbed(s *discordgo.Session) {
	// Get bot's profile picture
	botUser, err := s.User("@me")
	var avatarURL string
	if err == nil && botUser.Avatar != "" {
		avatarURL = discordgo.EndpointUserAvatar(botUser.ID, botUser.Avatar)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Partners",
		Description: "Click the below reactions to learn more about our partners, see what offerings they have for you, and how you can access their platform with us.",
	}
	if avatarURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: avatarURL}
	}

	var buttons []discordgo.MessageComponent
	for _, p := range partners {
		buttons = append(buttons, discordgo.Button{
			Label:    p.Name,
			Emoji:    &discordgo.ComponentEmoji{Name: p.Emoji},
			CustomID: "partner_" + p.Name,
			Style:    discordgo.PrimaryButton,
		})
	}

	if len(buttons) == 0 {
		s.ChannelMessageSend(partnersChannelID, "No partners configured yet.")
		return
	}

	_, err = s.ChannelMessageSendComplex(partnersChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}},
	})
	if err != nil {
		log.Printf("Error sending partners embed: %v", err)
	}
}

// --- End Partner Feature Section ---

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

	// Load partners from file
	if err := loadPartners(); err != nil {
		log.Fatalf("Error loading partners: %v", err)
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

	// --- Only send embed if not already present ---
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
	// ------------------------------------------------

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

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "addpartner",
			Description: "Add a new partner.",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name",        Description: "Partner Name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "description", Description: "Description",  Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "offering",    Description: "Offering",     Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "logo",        Description: "Logo URL",     Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "link",        Description: "Offering Link",Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "emoji",       Description: "Emoji",        Required: true},
			},
		},
	}

	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(s.State.User.ID, config.GuildID, cmd); err != nil {
			log.Printf("Error registering command '%s': %v", cmd.Name, err)
			continue
		}
		log.Printf("Registered command '%s' successfully.", cmd.Name)
	}
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

// Handle interactions for slash commands and partner buttons
func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle addpartner slash command
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "addpartner" {
		// Role restriction
		hasRole := false
		for _, r := range i.Member.Roles {
			if r == addPartnerRoleID {
				hasRole = true
				break
			}
		}
		if !hasRole {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You do not have permission to use this command.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		opts := i.ApplicationCommandData().Options
		p := Partner{
			Name:        opts[0].StringValue(),
			Description: opts[1].StringValue(),
			Offering:    opts[2].StringValue(),
			LogoURL:     opts[3].StringValue(),
			Link:        opts[4].StringValue(),
			Emoji:       opts[5].StringValue(),
		}
		partners = append(partners, p)
		if err := savePartners(); err != nil {
			log.Printf("Error saving partners: %v", err)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Partner added!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		// Optionally, update the partners embed after adding
		// sendPartnersEmbed(s)
		return
	}

	// Handle partner button presses
	if i.Type == discordgo.InteractionMessageComponent && strings.HasPrefix(i.MessageComponentData().CustomID, "partner_") {
		partnerName := strings.TrimPrefix(i.MessageComponentData().CustomID, "partner_")
		var p *Partner
		for idx, partner := range partners {
			if partner.Name == partnerName {
				p = &partners[idx]
				break
			}
		}
		if p == nil {
			return
		}
		hasRole := false
		for _, r := range i.Member.Roles {
			if r == accessRoleID {
				hasRole = true
				break
			}
		}
		embed := &discordgo.MessageEmbed{
			Title:       p.Name,
			Description: fmt.Sprintf("%s\n\n**Offering:** %s", p.Description, p.Offering),
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: p.LogoURL},
		}
		if hasRole {
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Access Offering", Value: fmt.Sprintf("[Click here](%s)", p.Link), Inline: false},
			}
		} else {
			embed.Fields = []*discordgo.MessageEmbedField{
				{Name: "Access Offering", Value: "Join BW4E to access all of our partner offerings.", Inline: false},
			}
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Ignore all other slash commands and interactions
}
