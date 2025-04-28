package main

import (
	"encoding/json"
	"os"
	"log"
	"github.com/bwmarrin/discordgo"
)

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

// Load partners from file
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

// Save partners to file
func savePartners() error {
	b, err := json.MarshalIndent(partners, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(partnersFile, b, 0644)
}

// Check for existing bot embed in the channel
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

// Send the partners embed with buttons
func sendPartnersEmbed(s *discordgo.Session) {
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
		name, id, animated := parseEmoji(p.Emoji)
		buttons = append(buttons, discordgo.Button{
			Label:    p.Name,
			Emoji:    &discordgo.ComponentEmoji{Name: name, ID: id, Animated: animated},
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

// Update the partners embed/buttons in-place
func updatePartnersEmbed(s *discordgo.Session, botUserID string) {
	messages, err := s.ChannelMessages(partnersChannelID, 50, "", "", "")
	if err != nil {
		log.Printf("Error fetching messages for update: %v", err)
		return
	}
	var botMsg *discordgo.Message
	for _, msg := range messages {
		if msg.Author != nil && msg.Author.ID == botUserID && len(msg.Embeds) > 0 {
			botMsg = msg
			break
		}
	}
	if botMsg == nil {
		sendPartnersEmbed(s)
		return
	}

	botUser, _ := s.User("@me")
	var avatarURL string
	if botUser.Avatar != "" {
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
		name, id, animated := parseEmoji(p.Emoji)
		buttons = append(buttons, discordgo.Button{
			Label:    p.Name,
			Emoji:    &discordgo.ComponentEmoji{Name: name, ID: id, Animated: animated},
			CustomID: "partner_" + p.Name,
			Style:    discordgo.PrimaryButton,
		})
	}

	edit := &discordgo.MessageEdit{
		ID:      botMsg.ID,
		Channel: partnersChannelID,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
	}
	if len(buttons) > 0 {
		edit.Components = &[]discordgo.MessageComponent{
			discordgo.ActionsRow{Components: buttons},
		}
	}

	_, err = s.ChannelMessageEditComplex(edit)
	if err != nil {
		log.Printf("Error editing partners embed: %v", err)
	}
}

// Partner slash command and button logic should be handled in handlers.go or onInteractionCreate,
// but you can also put helper functions here if needed.
