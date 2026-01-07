package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/app"
	"github.com/robertguss/bmad-automate-go/internal/config"
)

func main() {
	// Initialize configuration
	cfg := config.New()

	// Create the application model
	model := app.New(cfg)

	// Create the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running BMAD Automate: %v\n", err)
		os.Exit(1)
	}
}
