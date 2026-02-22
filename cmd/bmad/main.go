package main

import (
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/robertguss/bmad-automate-go/internal/app"
	"github.com/robertguss/bmad-automate-go/internal/config"
)

func main() {
	// Capture panic stack traces
	defer func() {
		if r := recover(); r != nil {
			// Write stack trace to file for debugging
			f, _ := os.Create("bmad-panic.log")
			if f != nil {
				fmt.Fprintf(f, "Panic: %v\n\nStack trace:\n%s", r, debug.Stack())
				f.Close()
			}
			fmt.Printf("Panic occurred: %v\nStack trace written to bmad-panic.log\n", r)
			os.Exit(1)
		}
	}()

	// Parse flags
	verbose := false
	var positionalArgs []string
	for _, arg := range os.Args[1:] {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Handle optional path argument
	if len(positionalArgs) > 0 {
		targetPath := positionalArgs[0]
		if err := os.Chdir(targetPath); err != nil {
			fmt.Printf("Error: cannot change to directory %q: %v\n", targetPath, err)
			os.Exit(1)
		}
	}

	// Initialize configuration
	cfg := config.New()
	if verbose {
		cfg.Verbose = true
	}

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
