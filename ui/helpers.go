package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderNavigationBar creates a consistent navigation bar for the bottom of the screen
func renderNavigationBar(width int, items map[string]string) string {
	// Handle invalid width
	if width <= 0 {
		width = 80 // Default fallback width
	}

	// Style for the navigation bar
	navBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(width).
		Align(lipgloss.Left)

	// Style for keys (keyboard shortcuts)
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFCC00"))

	// Style for descriptions
	descStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF"))

	// Build the navigation text with proper formatting
	var navTexts []string
	for key, desc := range items {
		navItem := fmt.Sprintf("%s %s",
			keyStyle.Render(key),
			descStyle.Render(desc))
		navTexts = append(navTexts, navItem)
	}

	// Join nav items with separators
	navText := strings.Join(navTexts, " | ")

	// Make sure it fits within the width
	if lipgloss.Width(navText) > width {
		// If too wide, only show the keys
		navTexts = []string{}
		for key := range items {
			navTexts = append(navTexts, keyStyle.Render(key))
		}
		navText = strings.Join(navTexts, " ")
	}

	// Render the full navigation bar
	return navBarStyle.Render(navText)
}
