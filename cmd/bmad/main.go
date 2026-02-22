package main

import (
	"flag"
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
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.BoolVar(verbose, "v", false, "Enable verbose output (shorthand)")
	apiEnabled := flag.Bool("api", false, "Enable REST API server")
	apiPort := flag.Int("port", config.DefaultAPIPort, "API server port")
	flag.Parse()

	// Handle optional path argument
	if flag.NArg() > 0 {
		targetPath := flag.Arg(0)
		if err := os.Chdir(targetPath); err != nil {
			fmt.Printf("Error: cannot change to directory %q: %v\n", targetPath, err)
			os.Exit(1)
		}
	}

	// Initialize configuration
	cfg := config.New()
	if *verbose {
		cfg.Verbose = true
	}
	if *apiEnabled {
		cfg.APIEnabled = true
	}
	if *apiPort != config.DefaultAPIPort {
		cfg.APIPort = *apiPort
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
