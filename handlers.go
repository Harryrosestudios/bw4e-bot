package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Handle interactions for slash commands and partner/notification buttons
func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// --- Partner commands ---
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "addpartner" {
		opts := i.ApplicationCommandData().Options
		newName := strings.TrimSpace(opts[0].StringValue())
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
		for _, p := range partners {
			if strings.EqualFold(strings.TrimSpace(p.Name), newName) {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "A partner with that name already exists. Please choose a unique name.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}
		p := Partner{
			Name:        newName,
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
		botUser, _ := s.User("@me")
		messages, err := s.ChannelMessages(partnersChannelID, 50, "", "", "")
		if err == nil {
			for _, msg := range messages {
				if msg.Author != nil && msg.Author.ID == botUser.ID && msg.Content == "No partners configured yet." {
					_ = s.ChannelMessageDelete(partnersChannelID, msg.ID)
					break
				}
			}
		}
		updatePartnersEmbed(s, botUser.ID)
		return
	}
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "delpartner" {
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
		partnerName := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
		found := false
		for idx, p := range partners {
			if strings.EqualFold(strings.TrimSpace(p.Name), partnerName) {
				partners = append(partners[:idx], partners[idx+1:]...)
				found = true
				break
			}
		}
		if !found {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Partner not found.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		if err := savePartners(); err != nil {
			log.Printf("Error saving partners: %v", err)
		}
		botUser, _ := s.User("@me")
		messages, err := s.ChannelMessages(partnersChannelID, 50, "", "", "")
		if err == nil {
			for _, msg := range messages {
				if msg.Author != nil && msg.Author.ID == botUser.ID && len(msg.Embeds) > 0 {
					_ = s.ChannelMessageDelete(partnersChannelID, msg.ID)
					break
				}
			}
		}
		sendPartnersEmbed(s)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Partner deleted and embed refreshed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// --- Notification channel commands ---
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "addnotificationchannel" {
		opts := i.ApplicationCommandData().Options
		name := opts[0].StringValue()
		channelID := parseID(opts[1].StringValue())
		roleID := parseID(opts[2].StringValue())
		notificationRoleID := parseID(opts[3].StringValue())
		for _, nc := range notificationChannels {
			if strings.EqualFold(nc.Name, name) {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "A notification channel with that name already exists.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
		}
		notificationChannels = append(notificationChannels, NotificationChannel{
			Name: name, ChannelID: channelID, AccessRoleID: roleID, NotificationRoleID: notificationRoleID,
		})
		saveNotificationChannels()
		sendNotificationsEmbed(s)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Notification channel added.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if i.Type == discordgo.InteractionApplicationCommand && i.ApplicationCommandData().Name == "delnotificationchannel" {
		name := i.ApplicationCommandData().Options[0].StringValue()
		found := false
		for idx, nc := range notificationChannels {
			if strings.EqualFold(nc.Name, name) {
				notificationChannels = append(notificationChannels[:idx], notificationChannels[idx+1:]...)
				found = true
				break
			}
		}
		if !found {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Notification channel not found.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		saveNotificationChannels()
		sendNotificationsEmbed(s)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Notification channel deleted.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// --- Notification button logic ---
	if i.Type == discordgo.InteractionMessageComponent && strings.HasPrefix(i.MessageComponentData().CustomID, "notification_") &&
		!strings.HasPrefix(i.MessageComponentData().CustomID, "notification_sub_") {

		name := strings.TrimPrefix(i.MessageComponentData().CustomID, "notification_")
		var nc *NotificationChannel
		for idx, n := range notificationChannels {
			if n.Name == name {
				nc = &notificationChannels[idx]
				break
			}
		}
		if nc == nil {
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Notification Subscription",
			Description: fmt.Sprintf("How would you like to subscribe to %s?", nc.Name),
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Subscribe with Notifications",
						CustomID: "notification_sub_with_" + nc.Name,
						Style:    discordgo.SuccessButton,
						Emoji:    &discordgo.ComponentEmoji{Name: "🔔"},
					},
					discordgo.Button{
						Label:    "Subscribe without Notifications",
						CustomID: "notification_sub_without_" + nc.Name,
						Style:    discordgo.SecondaryButton,
						Emoji:    &discordgo.ComponentEmoji{Name: "🔕"},
					},
				},
			},
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Flags:      discordgo.MessageFlagsEphemeral,
				Components: components,
			},
		})
		return
	}

	if i.Type == discordgo.InteractionMessageComponent && strings.HasPrefix(i.MessageComponentData().CustomID, "notification_sub_") {
		customID := i.MessageComponentData().CustomID
		var withNotifs bool
		var name string
		if strings.HasPrefix(customID, "notification_sub_with_") {
			withNotifs = true
			name = strings.TrimPrefix(customID, "notification_sub_with_")
		} else if strings.HasPrefix(customID, "notification_sub_without_") {
			withNotifs = false
			name = strings.TrimPrefix(customID, "notification_sub_without_")
		} else {
			return
		}

		var nc *NotificationChannel
		for idx, n := range notificationChannels {
			if n.Name == name {
				nc = &notificationChannels[idx]
				break
			}
		}
		if nc == nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Notification channel not found.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		if i.Member == nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Error: Could not determine your Discord member info.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		userID := i.Member.User.ID
		guildID := config.GuildID
		var err1, err2 error
		if withNotifs {
			err1 = s.GuildMemberRoleAdd(guildID, userID, nc.AccessRoleID)
			err2 = s.GuildMemberRoleAdd(guildID, userID, nc.NotificationRoleID)
		} else {
			err1 = s.GuildMemberRoleAdd(guildID, userID, nc.AccessRoleID)
			err2 = s.GuildMemberRoleRemove(guildID, userID, nc.NotificationRoleID)
		}
		if err1 != nil || err2 != nil {
			log.Printf("Role assignment error: %v %v", err1, err2)
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Failed to assign roles. Please contact an admin.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		emoji := "🔔"
		mode := "with notifications"
		if !withNotifs {
			emoji = "🔕"
			mode = "without notifications"
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Subscription Updated",
			Description: fmt.Sprintf("%s You are now subscribed to %s %s.", emoji, nc.Name, mode),
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

	// --- Partner button logic ---
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
		{
			Name:        "delpartner",
			Description: "Delete a partner by name.",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Partner Name", Required: true},
			},
		},
		{
			Name:        "addnotificationchannel",
			Description: "Add a notification channel.",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Channel Name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "channel", Description: "Channel (# or ID)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "role", Description: "Access Role (@ or ID)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "notificationrole", Description: "Notification Role (@ or ID)", Required: true},
			},
		},
		{
			Name:        "delnotificationchannel",
			Description: "Delete a notification channel by name.",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Channel Name", Required: true},
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
