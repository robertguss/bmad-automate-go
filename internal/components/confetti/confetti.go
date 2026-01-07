package confetti

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// Particle represents a single confetti particle
type Particle struct {
	X, Y     float64
	VelX     float64
	VelY     float64
	Char     string
	Color    lipgloss.Color
	Lifetime int
}

// TickMsg triggers animation frame update
type TickMsg time.Time

// Model represents the confetti animation
type Model struct {
	width     int
	height    int
	particles []Particle
	active    bool
	duration  int // frames remaining
	styles    theme.Styles
}

// Confetti characters
var confettiChars = []string{"*", "+", ".", "o", "x", "~", "^"}

// New creates a new confetti model
func New() Model {
	return Model{
		styles: theme.NewStyles(),
	}
}

// Start triggers the confetti animation
func (m *Model) Start(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.active = true
	m.duration = 60 // ~2 seconds at 30fps
	m.particles = m.generateParticles(50)
	return m.tick()
}

// Stop stops the confetti animation
func (m *Model) Stop() {
	m.active = false
	m.particles = nil
}

// IsActive returns whether the animation is running
func (m Model) IsActive() bool {
	return m.active
}

// SetSize updates the dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) generateParticles(count int) []Particle {
	t := theme.Current
	colors := []lipgloss.Color{
		t.Success,
		t.Primary,
		t.Secondary,
		t.Accent,
		t.Warning,
		t.Info,
	}

	particles := make([]Particle, count)
	for i := range particles {
		particles[i] = Particle{
			X:        float64(rand.Intn(m.width)),
			Y:        float64(rand.Intn(5)), // Start near top
			VelX:     (rand.Float64() - 0.5) * 2,
			VelY:     rand.Float64()*0.5 + 0.3,
			Char:     confettiChars[rand.Intn(len(confettiChars))],
			Color:    colors[rand.Intn(len(colors))],
			Lifetime: 60 + rand.Intn(30),
		}
	}
	return particles
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*33, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Init initializes the confetti model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the confetti animation
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		if !m.active {
			return m, nil
		}

		m.duration--
		if m.duration <= 0 {
			m.active = false
			m.particles = nil
			return m, nil
		}

		// Update particles
		var alive []Particle
		for i := range m.particles {
			p := &m.particles[i]
			p.X += p.VelX
			p.Y += p.VelY
			p.VelY += 0.05 // gravity
			p.Lifetime--

			// Keep particles that are still visible and alive
			if p.Y < float64(m.height) && p.Lifetime > 0 {
				alive = append(alive, *p)
			}
		}
		m.particles = alive

		return m, m.tick()
	}
	return m, nil
}

// View renders the confetti overlay
func (m Model) View() string {
	if !m.active || len(m.particles) == 0 {
		return ""
	}

	// Create a grid for rendering
	grid := make([][]string, m.height)
	for i := range grid {
		grid[i] = make([]string, m.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	// Place particles on grid
	for _, p := range m.particles {
		x := int(p.X)
		y := int(p.Y)
		if x >= 0 && x < m.width && y >= 0 && y < m.height {
			style := lipgloss.NewStyle().Foreground(p.Color)
			grid[y][x] = style.Render(p.Char)
		}
	}

	// Build output
	var rows []string
	for _, row := range grid {
		rows = append(rows, strings.Join(row, ""))
	}

	return strings.Join(rows, "\n")
}

// Overlay renders confetti over existing content
func (m Model) Overlay(content string, width, height int) string {
	if !m.active || len(m.particles) == 0 {
		return content
	}

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Ensure we have enough lines
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	// Overlay particles onto content
	for _, p := range m.particles {
		x := int(p.X)
		y := int(p.Y)
		if x >= 0 && y >= 0 && y < len(lines) {
			line := []rune(lines[y])
			// Ensure line is wide enough
			for len(line) <= x {
				line = append(line, ' ')
			}
			if x < len(line) {
				style := lipgloss.NewStyle().Foreground(p.Color)
				// Replace character at position
				runeChar := []rune(p.Char)[0]
				line[x] = []rune(style.Render(string(runeChar)))[0]
			}
			lines[y] = string(line)
		}
	}

	return strings.Join(lines, "\n")
}
