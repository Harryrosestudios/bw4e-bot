package main

import (
	"encoding/json"
	"os"

	"github.com/bwmarrin/discordgo"
)

// --- Notification Feature Section ---

type NotificationChannel struct {
	Name               string `json:"name"`
	ChannelID          string `json:"channel_id"`
	AccessRoleID       string `json:"access_role_id"`
	NotificationRoleID string `json:"notification_role_id"`
}

var notificationChannels []NotificationChannel
const notificationChannelsFile = "notification_channels.json"
const notificationsChannelID = "1366043181925273731"

// Load notification channels from file
func loadNotificationChannels() error {
	b, err := os.ReadFile(notificationChannelsFile)
	if err != nil {
		if os.IsNotExist(err) {
			notificationChannels = []NotificationChannel{}
			return nil
		}
		return err
	}
	return json.Unmarshal(b, &notificationChannels)
}

// Save notification channels to file
func saveNotificationChannels() error {
	b, err := json.MarshalIndent(notificationChannels, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(notificationChannelsFile, b, 0644)
}

// Send the notifications embed with buttons
func sendNotificationsEmbed(s *discordgo.Session) {
	embed := &discordgo.MessageEmbed{
		Title:       "Notifications",
		Description: "Use the reaction buttons below to gain access to our notification channels.",
	}
	var buttons []discordgo.MessageComponent
	for _, nc := range notificationChannels {
		buttons = append(buttons, discordgo.Button{
			Label:    nc.Name,
			CustomID: "notification_" + nc.Name,
			Style:    discordgo.PrimaryButton,
		})
	}
	if len(buttons) == 0 {
		s.ChannelMessageSend(notificationsChannelID, "No notification channels configured yet.")
		return
	}
	s.ChannelMessageSendComplex(notificationsChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}},
	})
}
