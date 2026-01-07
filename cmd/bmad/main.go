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

	// Set the program on the model's executor so it can send async messages
	// Note: This works because executor is a pointer, so even though model
	// was copied when passed to NewProgram, both copies share the same executor
	model.SetProgram(p)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running BMAD Automate: %v\n", err)
		os.Exit(1)
	}
}
