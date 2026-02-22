package markdown

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/robertguss/bmad-automate-go/internal/theme"
	"github.com/stretchr/testify/assert"
)

func TestParseInline_BoldSimple(t *testing.T) {
	segs := ParseInline("hello **world**")
	assert.Len(t, segs, 2)
	assert.Equal(t, Segment{Content: "hello ", Type: Text}, segs[0])
	assert.Equal(t, Segment{Content: "world", Type: Bold}, segs[1])
}

func TestParseInline_CodeSimple(t *testing.T) {
	segs := ParseInline("run `go test`")
	assert.Len(t, segs, 2)
	assert.Equal(t, Segment{Content: "run ", Type: Text}, segs[0])
	assert.Equal(t, Segment{Content: "go test", Type: Code}, segs[1])
}

func TestParseInline_Mixed(t *testing.T) {
	segs := ParseInline("**bold** then `code` end")
	assert.Len(t, segs, 4)
	assert.Equal(t, Bold, segs[0].Type)
	assert.Equal(t, Text, segs[1].Type)
	assert.Equal(t, Code, segs[2].Type)
	assert.Equal(t, Text, segs[3].Type)
}

func TestParseInline_Unclosed(t *testing.T) {
	segs := ParseInline("hello **world")
	assert.Len(t, segs, 1)
	assert.Equal(t, Segment{Content: "hello **world", Type: Text}, segs[0])
}

func TestParseInline_UnclosedBacktick(t *testing.T) {
	segs := ParseInline("hello `world")
	assert.Len(t, segs, 1)
	assert.Equal(t, Segment{Content: "hello `world", Type: Text}, segs[0])
}

func TestParseInline_Empty(t *testing.T) {
	segs := ParseInline("")
	assert.Empty(t, segs)
}

func TestParseInline_PlainText(t *testing.T) {
	segs := ParseInline("just plain text")
	assert.Len(t, segs, 1)
	assert.Equal(t, Segment{Content: "just plain text", Type: Text}, segs[0])
}

// classifyLine tests

func TestClassifyLine_H1(t *testing.T) {
	kind, _, body := classifyLine("# Title")
	assert.Equal(t, kindHeader, kind)
	assert.Equal(t, "Title", body)
}

func TestClassifyLine_H2(t *testing.T) {
	kind, _, body := classifyLine("## My Header")
	assert.Equal(t, kindHeader, kind)
	assert.Equal(t, "My Header", body)
}

func TestClassifyLine_H3(t *testing.T) {
	kind, _, body := classifyLine("### Step 9")
	assert.Equal(t, kindHeader, kind)
	assert.Equal(t, "Step 9", body)
}

func TestClassifyLine_H4(t *testing.T) {
	kind, _, body := classifyLine("#### Details")
	assert.Equal(t, kindHeader, kind)
	assert.Equal(t, "Details", body)
}

func TestClassifyLine_Bullet(t *testing.T) {
	kind, prefix, body := classifyLine("- item one")
	assert.Equal(t, kindBullet, kind)
	assert.Equal(t, "- ", prefix)
	assert.Equal(t, "item one", body)
}

func TestClassifyLine_StarBullet(t *testing.T) {
	kind, prefix, body := classifyLine("* item two")
	assert.Equal(t, kindBullet, kind)
	assert.Equal(t, "* ", prefix)
	assert.Equal(t, "item two", body)
}

func TestClassifyLine_NumberedList(t *testing.T) {
	kind, prefix, body := classifyLine("1. first item")
	assert.Equal(t, kindNumbered, kind)
	assert.Equal(t, "1. ", prefix)
	assert.Equal(t, "first item", body)
}

func TestClassifyLine_NumberedListMultiDigit(t *testing.T) {
	kind, prefix, body := classifyLine("12. twelfth item")
	assert.Equal(t, kindNumbered, kind)
	assert.Equal(t, "12. ", prefix)
	assert.Equal(t, "twelfth item", body)
}

func TestClassifyLine_Rule(t *testing.T) {
	kind, _, _ := classifyLine("---")
	assert.Equal(t, kindRule, kind)
}

func TestClassifyLine_Plain(t *testing.T) {
	kind, prefix, body := classifyLine("normal text")
	assert.Equal(t, kindPlain, kind)
	assert.Equal(t, "", prefix)
	assert.Equal(t, "normal text", body)
}

// RenderLine tests

func TestRenderLine_Empty(t *testing.T) {
	result := RenderLine("", false, theme.Catppuccin)
	assert.Equal(t, "", result)
}

func TestRenderLine_BoldStripped(t *testing.T) {
	result := RenderLine("hello **world**", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
	stripped := ansi.Strip(result)
	assert.NotContains(t, stripped, "**")
	assert.Contains(t, stripped, "world")
}

func TestRenderLine_CodeStripped(t *testing.T) {
	result := RenderLine("run `go test`", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
	stripped := ansi.Strip(result)
	assert.NotContains(t, stripped, "`")
	assert.Contains(t, stripped, "go test")
}

func TestRenderLine_Stderr(t *testing.T) {
	result := RenderLine("error msg", true, theme.Catppuccin)
	assert.NotEmpty(t, result)
}

func TestRenderLine_H2(t *testing.T) {
	result := RenderLine("## Title", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Equal(t, "Title", stripped)
}

func TestRenderLine_H3(t *testing.T) {
	result := RenderLine("### Step 9 : Completion", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Equal(t, "Step 9 : Completion", stripped)
	assert.NotContains(t, stripped, "#")
}

func TestRenderLine_Bullet(t *testing.T) {
	result := RenderLine("- item with **bold**", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.NotContains(t, stripped, "**")
	assert.Contains(t, stripped, "bold")
}

func TestRenderLine_HorizontalRule(t *testing.T) {
	result := RenderLine("---", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Contains(t, stripped, "─")
}

func TestRenderLine_NumberedList(t *testing.T) {
	result := RenderLine("1. first item", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Contains(t, stripped, "1. ")
	assert.Contains(t, stripped, "first item")
}

func TestRenderLine_Emoji(t *testing.T) {
	result := RenderLine("done! 🎉", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
}

func TestRenderLine_WithAnsiInput(t *testing.T) {
	// Simulate ANSI-styled input from Claude CLI
	input := "\x1b[1m**bold text**\x1b[0m"
	result := RenderLine(input, false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.NotContains(t, stripped, "**")
	assert.Contains(t, stripped, "bold text")
}

func TestRenderLine_AnsiAroundBackticks(t *testing.T) {
	input := "run \x1b[36m`go test`\x1b[0m please"
	result := RenderLine(input, false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.NotContains(t, stripped, "`")
	assert.Contains(t, stripped, "go test")
}

// Table tests

func TestClassifyLine_TableRow(t *testing.T) {
	kind, _, _ := classifyLine("| Élément | Statut |")
	assert.Equal(t, kindTable, kind)
}

func TestClassifyLine_TableSeparator(t *testing.T) {
	kind, _, _ := classifyLine("|---|---|")
	assert.Equal(t, kindTableSep, kind)
}

func TestClassifyLine_TableSeparatorSpaced(t *testing.T) {
	kind, _, _ := classifyLine("| --- | --- |")
	assert.Equal(t, kindTableSep, kind)
}

func TestClassifyLine_TableSeparatorAligned(t *testing.T) {
	kind, _, _ := classifyLine("|:---|---:|")
	assert.Equal(t, kindTableSep, kind)
}

func TestIsTableSeparator(t *testing.T) {
	assert.True(t, isTableSeparator("|---|---|"))
	assert.True(t, isTableSeparator("| --- | --- |"))
	assert.True(t, isTableSeparator("|:---|:---:|"))
	assert.False(t, isTableSeparator("| hello | world |"))
	assert.False(t, isTableSeparator("|"))
}

func TestSplitTableCells(t *testing.T) {
	cells := splitTableCells("| a | b | c |")
	assert.Len(t, cells, 3)
	assert.Equal(t, " a ", cells[0])
	assert.Equal(t, " b ", cells[1])
	assert.Equal(t, " c ", cells[2])
}

func TestRenderLine_TableRow(t *testing.T) {
	result := RenderLine("| Tests | 35 passent, 0 échecs |", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Contains(t, stripped, "│")
	assert.Contains(t, stripped, "Tests")
	assert.Contains(t, stripped, "35 passent")
	assert.NotContains(t, stripped, "|") // raw pipes replaced by │
}

func TestRenderLine_TableRowWithBold(t *testing.T) {
	result := RenderLine("| Statut story | **done** |", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Contains(t, stripped, "done")
	assert.NotContains(t, stripped, "**")
}

func TestRenderLine_TableSeparator(t *testing.T) {
	result := RenderLine("|---|---|", false, theme.Catppuccin)
	stripped := ansi.Strip(result)
	assert.Contains(t, stripped, "├")
	assert.Contains(t, stripped, "─")
	assert.Contains(t, stripped, "┤")
}
