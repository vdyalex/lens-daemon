package latex_test

import (
	"testing"

	"github.com/vdyalex/lens-daemon/src/helpers/latex"
)

func TestToUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text passthrough",
			input:    "hello world",
			expected: "hello world",
		},

		// Delimiter stripping
		{
			name:     "inline dollar delimiters",
			input:    "$x + y$",
			expected: "x + y",
		},
		{
			name:     "display dollar delimiters",
			input:    "$$x + y$$",
			expected: "x + y",
		},
		{
			name:     "inline paren delimiters",
			input:    `\(x + y\)`,
			expected: "x + y",
		},
		{
			name:     "display bracket delimiters",
			input:    `\[x + y\]`,
			expected: "x + y",
		},

		// Greek letters
		{
			name:     "greek lowercase",
			input:    `$\alpha + \beta$`,
			expected: "α + β",
		},
		{
			name:     "greek uppercase",
			input:    `\Omega and \Delta`,
			expected: "Ω and Δ",
		},

		// Math operators
		{
			name:     "inequality operators",
			input:    `x \neq y, a \leq b, c \geq d`,
			expected: "x ≠ y, a ≤ b, c ≥ d",
		},
		{
			name:     "arithmetic operators",
			input:    `a \pm b \times c \div d`,
			expected: "a ± b × c ÷ d",
		},
		{
			name:     "sum and integral",
			input:    `\sum and \int`,
			expected: "∑ and ∫",
		},
		{
			name:     "infinity",
			input:    `\infty`,
			expected: "∞",
		},

		// Arrows
		{
			name:     "arrows",
			input:    `A \rightarrow B \Rightarrow C`,
			expected: "A → B ⇒ C",
		},

		// Set and logic
		{
			name:     "set operators",
			input:    `x \in A \cup B`,
			expected: "x ∈ A ∪ B",
		},
		{
			name:     "logic operators",
			input:    `p \land q \lor r`,
			expected: "p ∧ q ∨ r",
		},

		// Superscripts
		{
			name:     "single superscript",
			input:    "x^2",
			expected: "x²",
		},
		{
			name:     "braced superscript",
			input:    "x^{10}",
			expected: "x¹⁰",
		},
		{
			name:     "superscript n",
			input:    "x^n",
			expected: "xⁿ",
		},

		// Subscripts
		{
			name:     "single subscript",
			input:    "x_0",
			expected: "x₀",
		},
		{
			name:     "braced subscript",
			input:    "x_{12}",
			expected: "x₁₂",
		},

		// Fractions
		{
			name:     "simple fraction",
			input:    `\frac{a}{b}`,
			expected: "a/b",
		},
		{
			name:     "fraction with expressions",
			input:    `\frac{x+1}{y-1}`,
			expected: "x+1/y-1",
		},

		// Square root
		{
			name:     "sqrt with braces",
			input:    `\sqrt{x}`,
			expected: "√(x)",
		},
		{
			name:     "sqrt with expression",
			input:    `\sqrt{x+1}`,
			expected: "√(x+1)",
		},

		// Mixed expressions
		{
			name:     "quadratic formula simplified",
			input:    `$x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$`,
			expected: "x = -b ± √(b² - 4ac)/2a",
		},
		{
			name:     "summation with bounds",
			input:    `\sum_{i=0}^{n}`,
			expected: "∑ᵢ₌₀ⁿ",
		},
		{
			name:     "nested braces in fraction",
			input:    `\frac{\sqrt{x}}{y}`,
			expected: "√(x)/y",
		},

		// Word boundary
		{
			name:     "command not matched inside word",
			input:    `\integral`,
			expected: `\integral`,
		},
		{
			name:     "command at end of string",
			input:    `value is \pi`,
			expected: "value is π",
		},
		{
			name:     "command followed by punctuation",
			input:    `\alpha.`,
			expected: "α.",
		},

		// Edge cases
		{
			name:     "trailing backslash",
			input:    `test\`,
			expected: `test\`,
		},
		{
			name:     "lone dollar sign removed",
			input:    "cost is $5",
			expected: "cost is 5",
		},
		{
			name:     "unknown command preserved",
			input:    `\unknowncmd`,
			expected: `\unknowncmd`,
		},
		{
			name:     "ellipsis",
			input:    `a_1, a_2, \ldots, a_n`,
			expected: "a₁, a₂, …, aₙ",
		},
		{
			name:     "unmapped superscript character",
			input:    "x^a",
			expected: "xa",
		},

		// Code preservation
		{
			name:     "inline code preserved",
			input:    "use `$variable` here",
			expected: "use `$variable` here",
		},
		{
			name:     "fenced code block preserved",
			input:    "text\n```\n\\alpha = $x\n```\nmore",
			expected: "text\n```\n\\alpha = $x\n```\nmore",
		},
		{
			name:     "latex outside code converted",
			input:    "`code` and \\alpha",
			expected: "`code` and α",
		},
		{
			name:     "mixed code and latex",
			input:    "\\beta is `\\beta` literally",
			expected: "β is `\\beta` literally",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latex.ToUnicode(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, expected %q", got, tt.expected)
			}
		})
	}
}
