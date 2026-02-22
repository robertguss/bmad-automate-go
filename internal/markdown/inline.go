package markdown

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/robertguss/bmad-automate-go/internal/theme"
)

// SegmentType identifies the kind of inline markup.
type SegmentType int

const (
	Text SegmentType = iota
	Bold
	Code
)

// Segment is a slice of a line with an associated style type.
type Segment struct {
	Content string
	Type    SegmentType
}

// ParseInline scans a line and splits it into typed segments.
// Unclosed markers are treated as literal text.
func ParseInline(line string) []Segment {
	var segments []Segment
	i := 0
	buf := strings.Builder{}

	flush := func(typ SegmentType) {
		if buf.Len() > 0 {
			segments = append(segments, Segment{Content: buf.String(), Type: typ})
			buf.Reset()
		}
	}

	for i < len(line) {
		// Bold: **...**
		if i+1 < len(line) && line[i] == '*' && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "**")
			if end >= 0 {
				flush(Text)
				segments = append(segments, Segment{Content: line[i+2 : i+2+end], Type: Bold})
				i += 2 + end + 2
				continue
			}
		}

		// Inline code: `...`
		if line[i] == '`' {
			end := strings.IndexByte(line[i+1:], '`')
			if end >= 0 {
				flush(Text)
				segments = append(segments, Segment{Content: line[i+1 : i+1+end], Type: Code})
				i += 1 + end + 1
				continue
			}
		}

		buf.WriteByte(line[i])
		i++
	}
	flush(Text)
	return segments
}

// lineKind classifies the structural role of a line.
type lineKind int

const (
	kindPlain     lineKind = iota
	kindBullet             // "- " prefix
	kindHeader             // "# "/"## "/"### "/etc.
	kindRule               // "---" horizontal rule
	kindNumbered           // "1. " numbered list
)

func classifyLine(line string) (lineKind, string, string) {
	// Headers: # through ####
	if strings.HasPrefix(line, "# ") {
		return kindHeader, "", line[2:]
	}
	if strings.HasPrefix(line, "## ") {
		return kindHeader, "", line[3:]
	}
	if strings.HasPrefix(line, "### ") {
		return kindHeader, "", line[4:]
	}
	if strings.HasPrefix(line, "#### ") {
		return kindHeader, "", line[5:]
	}

	// Horizontal rule
	if line == "---" || line == "***" || line == "___" {
		return kindRule, "", ""
	}

	// Bullet list
	if strings.HasPrefix(line, "- ") {
		return kindBullet, "- ", line[2:]
	}
	if strings.HasPrefix(line, "* ") {
		return kindBullet, "* ", line[2:]
	}

	// Numbered list (e.g. "1. ")
	for j := 0; j < len(line) && j < 4; j++ {
		if line[j] >= '0' && line[j] <= '9' {
			continue
		}
		if line[j] == '.' && j > 0 && j+1 < len(line) && line[j+1] == ' ' {
			return kindNumbered, line[:j+2], line[j+2:]
		}
		break
	}

	return kindPlain, "", line
}

// RenderLine applies inline markdown rendering to a single line.
// It strips ANSI escape codes first so that markdown markers are
// reliably detected even when the source emits styled output.
func RenderLine(line string, isStderr bool, t theme.Theme) string {
	if line == "" {
		return ""
	}

	// Strip any ANSI escape codes from the input so pattern matching works
	clean := ansi.Strip(line)

	kind, prefix, body := classifyLine(clean)

	var baseColor lipgloss.Color
	if isStderr {
		baseColor = t.Error
	} else {
		baseColor = t.Foreground
	}

	switch kind {
	case kindHeader:
		return lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Render(body)

	case kindRule:
		return lipgloss.NewStyle().Foreground(t.Subtle).Render("─────────────────────────────")

	case kindBullet:
		prefixStr := lipgloss.NewStyle().Foreground(t.Subtle).Render(prefix)
		return prefixStr + renderSegments(ParseInline(body), baseColor, t)

	case kindNumbered:
		prefixStr := lipgloss.NewStyle().Foreground(t.Subtle).Render(prefix)
		return prefixStr + renderSegments(ParseInline(body), baseColor, t)

	default:
		return renderSegments(ParseInline(body), baseColor, t)
	}
}

func renderSegments(segments []Segment, baseColor lipgloss.Color, t theme.Theme) string {
	var b strings.Builder
	for _, seg := range segments {
		switch seg.Type {
		case Bold:
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(baseColor).Render(seg.Content))
		case Code:
			b.WriteString(lipgloss.NewStyle().Foreground(t.Accent).Render(seg.Content))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(baseColor).Render(seg.Content))
		}
	}
	return b.String()
}
