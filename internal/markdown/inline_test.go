package markdown

import (
	"testing"

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

func TestClassifyLine_Header(t *testing.T) {
	kind, prefix, body := classifyLine("## My Header")
	assert.Equal(t, kindHeader, kind)
	assert.Equal(t, "", prefix)
	assert.Equal(t, "My Header", body)
}

func TestClassifyLine_Bullet(t *testing.T) {
	kind, prefix, body := classifyLine("- item one")
	assert.Equal(t, kindBullet, kind)
	assert.Equal(t, "- ", prefix)
	assert.Equal(t, "item one", body)
}

func TestClassifyLine_Plain(t *testing.T) {
	kind, prefix, body := classifyLine("normal text")
	assert.Equal(t, kindPlain, kind)
	assert.Equal(t, "", prefix)
	assert.Equal(t, "normal text", body)
}

func TestRenderLine_Empty(t *testing.T) {
	result := RenderLine("", false, theme.Catppuccin)
	assert.Equal(t, "", result)
}

func TestRenderLine_BoldNotEmpty(t *testing.T) {
	result := RenderLine("hello **world**", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "**")
}

func TestRenderLine_CodeNotEmpty(t *testing.T) {
	result := RenderLine("run `go test`", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "`")
}

func TestRenderLine_Stderr(t *testing.T) {
	result := RenderLine("error msg", true, theme.Catppuccin)
	assert.NotEmpty(t, result)
}

func TestRenderLine_Header(t *testing.T) {
	result := RenderLine("## Title", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
}

func TestRenderLine_Bullet(t *testing.T) {
	result := RenderLine("- item with **bold**", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "**")
}

func TestRenderLine_Emoji(t *testing.T) {
	result := RenderLine("done! 🎉", false, theme.Catppuccin)
	assert.NotEmpty(t, result)
}
