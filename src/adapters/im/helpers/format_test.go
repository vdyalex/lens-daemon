package helpers_test

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/adapters/im/helpers"
)

func TestToTelegramMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "reserved chars escaped",
			input:    "1.99!",
			expected: "1\\.99\\!",
		},
		{
			name:     "bold double asterisk",
			input:    "**bold**",
			expected: "*bold*",
		},
		{
			name:     "bold with reserved chars",
			input:    "**1.5x**",
			expected: "*1\\.5x*",
		},
		{
			name:     "italic asterisk",
			input:    "*it*",
			expected: "_it_",
		},
		{
			name:     "italic underscore",
			input:    "_it_",
			expected: "_it_",
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: "`code`",
		},
		{
			name:     "inline code with reserved chars",
			input:    "`a.b`",
			expected: "`a.b`",
		},
		{
			name:     "fenced code block with lang",
			input:    "```go\ncode\n```",
			expected: "```\ncode\n```",
		},
		{
			name:     "fenced code block no lang",
			input:    "```\ncode\n```",
			expected: "```\ncode\n```",
		},
		{
			name:     "header level 1",
			input:    "# Title",
			expected: "*Title*",
		},
		{
			name:     "header level 3",
			input:    "### Sub",
			expected: "*Sub*",
		},
		{
			name:     "header with reserved chars",
			input:    "# Hi!",
			expected: "*Hi\\!*",
		},
		{
			name:     "mid-line hash not header",
			input:    "text # foo",
			expected: "text \\# foo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "backslash escaped",
			input:    "\\",
			expected: "\\\\",
		},
		{
			name:     "unclosed bold",
			input:    "**bold",
			expected: "*bold",
		},
		{
			name:     "unclosed italic",
			input:    "*it",
			expected: "_it",
		},
		{
			name:     "unclosed inline code",
			input:    "`code",
			expected: "`code",
		},
		{
			name:     "multiple reserved chars",
			input:    "a.b!c-d",
			expected: "a\\.b\\!c\\-d",
		},
		{
			name:     "bold with inner italic",
			input:    "**bold *italic* here**",
			expected: "*bold \\*italic\\* here*",
		},
		{
			name:     "header with newline",
			input:    "# Header\nmore text",
			expected: "*Header*\nmore text",
		},
		{
			name:     "fenced code with reserved chars inside",
			input:    "```\na.b! c-d\n```",
			expected: "```\na.b! c-d\n```",
		},
		{
			name:     "multiple headers",
			input:    "# H1\nsome text\n## H2",
			expected: "*H1*\nsome text\n*H2*",
		},
		{
			name:     "mixed formatting",
			input:    "This is **bold** and *italic* text with `code`",
			expected: "This is *bold* and _italic_ text with `code`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helpers.ToTelegramMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, expected %q", got, tt.expected)
			}
		})
	}
}
