package latex

import (
	"sort"
	"strings"
	"unicode"
)

// ToUnicode replaces LaTeX math notation with Unicode equivalents and strips
// LaTeX delimiters ($...$, $$...$$, \[...\], \(...\)).
//
// Code spans (backtick-delimited) and fenced code blocks are preserved as-is.
// Handles Greek letters, math operators, superscripts, subscripts, and simple
// fractions. Designed for near-zero latency via string replacement.
//
// text: input string potentially containing LaTeX notation.
// Returns the input with LaTeX replaced by Unicode equivalents.
func ToUnicode(text string) string {
	if text == "" {
		return text
	}

	runes := []rune(text)
	var out strings.Builder

	for i := 0; i < len(runes); {
		// Fenced code block: ```...``` — preserve verbatim
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			i = copyUntilClosingFence(runes, i, &out)
			continue
		}

		// Inline code: `...` — preserve verbatim
		if runes[i] == '`' {
			i = copyUntilClosingBacktick(runes, i, &out)
			continue
		}

		// Accumulate non-code text for conversion
		start := i
		for i < len(runes) && runes[i] != '`' {
			i++
		}
		segment := string(runes[start:i])
		out.WriteString(convertSegment(segment))
	}

	return out.String()
}

// convertSegment applies all LaTeX-to-Unicode transformations to a non-code text segment.
func convertSegment(text string) string {
	result := stripDelimiters(text)
	result = convertFractions(result)
	result = convertSqrt(result)
	result = replaceCommands(result)
	result = convertScripts(result)

	return result
}

// copyUntilClosingFence copies a fenced code block (```...```) verbatim to out.
// Returns the position after the closing fence.
func copyUntilClosingFence(runes []rune, start int, out *strings.Builder) int {
	out.WriteString("```")
	i := start + 3
	for i < len(runes) {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			out.WriteString("```")
			return i + 3
		}
		out.WriteRune(runes[i])
		i++
	}

	return i
}

// copyUntilClosingBacktick copies an inline code span (`...`) verbatim to out.
// Returns the position after the closing backtick.
func copyUntilClosingBacktick(runes []rune, start int, out *strings.Builder) int {
	out.WriteRune('`')
	i := start + 1
	for i < len(runes) && runes[i] != '`' {
		out.WriteRune(runes[i])
		i++
	}
	if i < len(runes) {
		out.WriteRune('`')
		i++
	}

	return i
}

// stripDelimiters removes LaTeX math delimiters while preserving inner content.
// Processes display delimiters ($$, \[...\]) before inline ($, \(...\)).
func stripDelimiters(text string) string {
	result := strings.ReplaceAll(text, "$$", "")
	result = replaceDelimiterPair(result, `\[`, `\]`)
	result = replaceDelimiterPair(result, `\(`, `\)`)
	result = strings.ReplaceAll(result, "$", "")

	return result
}

// replaceDelimiterPair removes matched opening/closing delimiter pairs.
func replaceDelimiterPair(text, open, close string) string {
	result := strings.ReplaceAll(text, open, "")
	result = strings.ReplaceAll(result, close, "")

	return result
}

// convertFractions replaces \frac{numerator}{denominator} with numerator/denominator.
func convertFractions(text string) string {
	runes := []rune(text)
	var out strings.Builder
	prefix := []rune(`\frac{`)

	for i := 0; i < len(runes); {
		if matchesAt(runes, i, prefix) {
			i += len(prefix)
			numerator, end := readBraceContent(runes, i)
			i = end
			if i < len(runes) && runes[i] == '{' {
				i++
				denominator, end := readBraceContent(runes, i)
				i = end
				out.WriteString(numerator)
				out.WriteRune('/')
				out.WriteString(denominator)
				continue
			}
			out.WriteString(numerator)
			continue
		}
		out.WriteRune(runes[i])
		i++
	}

	return out.String()
}

// convertSqrt replaces \sqrt{content} with √(content).
// A bare \sqrt without braces is handled by replaceCommands.
func convertSqrt(text string) string {
	runes := []rune(text)
	var out strings.Builder
	prefix := []rune(`\sqrt{`)

	for i := 0; i < len(runes); {
		if matchesAt(runes, i, prefix) {
			i += len(prefix)
			content, end := readBraceContent(runes, i)
			i = end
			out.WriteString("√(")
			out.WriteString(content)
			out.WriteRune(')')
			continue
		}
		out.WriteRune(runes[i])
		i++
	}

	return out.String()
}

// convertScripts replaces ^{...}, ^x, _{...}, and _x with Unicode equivalents.
func convertScripts(text string) string {
	runes := []rune(text)
	var out strings.Builder

	for i := 0; i < len(runes); {
		isScript := (runes[i] == '^' || runes[i] == '_') && i+1 < len(runes)
		if !isScript {
			out.WriteRune(runes[i])
			i++
			continue
		}

		table := superscripts
		if runes[i] == '_' {
			table = subscripts
		}
		i++

		if runes[i] == '{' {
			i++
			for i < len(runes) && runes[i] != '}' {
				mapped, ok := table[runes[i]]
				if ok {
					out.WriteRune(mapped)
				} else {
					out.WriteRune(runes[i])
				}
				i++
			}
			if i < len(runes) {
				i++ // skip closing '}'
			}
		} else {
			mapped, ok := table[runes[i]]
			if ok {
				out.WriteRune(mapped)
			} else {
				out.WriteRune(runes[i])
			}
			i++
		}
	}

	return out.String()
}

// replaceCommands substitutes LaTeX \command sequences with Unicode symbols.
// Uses longest-match-first ordering with word-boundary detection.
func replaceCommands(text string) string {
	runes := []rune(text)
	var out strings.Builder

	for i := 0; i < len(runes); {
		if runes[i] != '\\' {
			out.WriteRune(runes[i])
			i++
			continue
		}

		replaced := false
		for _, key := range sortedCommandKeys {
			keyRunes := []rune(key)
			if !matchesAt(runes, i, keyRunes) {
				continue
			}
			end := i + len(keyRunes)
			isWordBoundary := end >= len(runes) || !unicode.IsLetter(runes[end])
			if !isWordBoundary {
				continue
			}
			out.WriteString(commands[key])
			i = end
			replaced = true
			break
		}
		if !replaced {
			out.WriteRune(runes[i])
			i++
		}
	}

	return out.String()
}

// matchesAt checks whether pattern appears in runes starting at position index.
func matchesAt(runes []rune, index int, pattern []rune) bool {
	if index+len(pattern) > len(runes) {
		return false
	}
	for j, p := range pattern {
		if runes[index+j] != p {
			return false
		}
	}

	return true
}

// readBraceContent reads characters until a matching '}' at the same nesting
// depth and returns the content and the position after the closing brace.
// Nested brace pairs are included in the content. If no closing brace is found,
// returns content up to end of input.
func readBraceContent(runes []rune, start int) (string, int) {
	var content strings.Builder
	depth := 0
	i := start
	for i < len(runes) {
		if runes[i] == '{' {
			depth++
		}
		if runes[i] == '}' {
			if depth == 0 {
				i++ // skip closing '}'
				break
			}
			depth--
		}
		content.WriteRune(runes[i])
		i++
	}

	return content.String(), i
}

// initSortedKeys returns the map keys sorted by descending length for greedy matching.
func initSortedKeys(source map[string]string) []string {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	return keys
}
