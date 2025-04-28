package main

import (
	"regexp"
	"strings"
)

// Helper for *int values in struct literals (for DiscordGo components)
func intPtr(i int) *int {
	return &i
}

// Helper: Parse a channel or role mention or ID into just the ID
func parseID(input string) string {
	// Handles <#channel>, <@&role>, <@role>, or raw IDs
	return strings.Trim(input, "<#@&!>")
}

// Emoji parsing helper
var customEmojiPattern = regexp.MustCompile(`<a?:(\w+):(\d+)>`)

func parseEmoji(input string) (name, id string, animated bool) {
	matches := customEmojiPattern.FindStringSubmatch(input)
	if len(matches) == 3 {
		animated := strings.HasPrefix(input, "<a:")
		return matches[1], matches[2], animated
	}
	// fallback: handle Unicode emoji
	return input, "", false
}
