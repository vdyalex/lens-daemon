package messenger

import "strings"

// reserved contains all MarkdownV2 characters that must be escaped with \
// when they appear outside of code spans.
var reserved = map[rune]bool{
	'_': true, '*': true, '[': true, ']': true, '(': true, ')': true,
	'~': true, '`': true, '>': true, '#': true, '+': true, '-': true,
	'=': true, '|': true, '{': true, '}': true, '.': true, '!': true, '\\': true,
}

// toTelegramMarkdown converts a standard-markdown string to Telegram MarkdownV2.
//
// Conversion rules:
//   - Fenced code blocks (```lang\n...\n```) → preserved; language hint stripped
//   - Inline code (`text`) → preserved as-is; content not escaped
//   - Bold (**text**) → *text* with inner chars escaped
//   - Italic (*text* or _text_) → _text_ with inner chars escaped
//   - Headers (# Heading) → *Heading* (bold) with inner chars escaped
//   - All other text → reserved characters escaped with \
//
// Note: markdown spans that cross a 4096-rune message chunk boundary will not
// render correctly; this is a known limitation of per-chunk formatting.
// TODO: split at formatting boundaries instead of rune count.
func toTelegramMarkdown(text string) string {
	runes := []rune(text)
	runeCount := len(runes)
	var out strings.Builder

	for i := 0; i < runeCount; {
		// Fenced code block: ```[lang]\n...\n```
		if i+2 < runeCount && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			out.WriteString("```")
			i += 3
			for i < runeCount && runes[i] != '\n' { // skip language hint
				i++
			}
			if i < runeCount {
				out.WriteRune('\n')
				i++
			}
			for i < runeCount {
				if i+2 < runeCount && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
					out.WriteString("```")
					i += 3
					break
				}
				out.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Inline code: `text`
		if runes[i] == '`' {
			out.WriteRune('`')
			i++
			for i < runeCount && runes[i] != '`' {
				out.WriteRune(runes[i])
				i++
			}
			if i < runeCount {
				out.WriteRune('`')
				i++
			}
			continue
		}

		// Bold: **text** → *text*
		if i+1 < runeCount && runes[i] == '*' && runes[i+1] == '*' {
			out.WriteRune('*')
			i += 2
			for i < runeCount && !(i+1 < runeCount && runes[i] == '*' && runes[i+1] == '*') {
				writeEscaped(&out, runes[i])
				i++
			}
			if i+1 < runeCount {
				out.WriteRune('*')
				i += 2
			}
			continue
		}

		// Italic: *text* (single asterisk) → _text_
		if runes[i] == '*' {
			out.WriteRune('_')
			i++
			for i < runeCount && runes[i] != '*' {
				writeEscaped(&out, runes[i])
				i++
			}
			if i < runeCount {
				out.WriteRune('_')
				i++
			}
			continue
		}

		// Italic: _text_ → _text_
		if runes[i] == '_' {
			out.WriteRune('_')
			i++
			for i < runeCount && runes[i] != '_' {
				writeEscaped(&out, runes[i])
				i++
			}
			if i < runeCount {
				out.WriteRune('_')
				i++
			}
			continue
		}

		// Header: # Heading → *Heading* (MarkdownV2 has no heading concept)
		if runes[i] == '#' && (i == 0 || runes[i-1] == '\n') {
			for i < runeCount && runes[i] == '#' {
				i++
			}
			for i < runeCount && runes[i] == ' ' {
				i++
			}
			out.WriteRune('*')
			for i < runeCount && runes[i] != '\n' {
				writeEscaped(&out, runes[i])
				i++
			}
			out.WriteRune('*')
			continue
		}

		// Plain text: escape all reserved characters
		writeEscaped(&out, runes[i])
		i++
	}

	return out.String()
}

// writeEscaped writes r to builder, prepending \ if r is a MarkdownV2 reserved character.
func writeEscaped(builder *strings.Builder, r rune) {
	if reserved[r] {
		builder.WriteRune('\\')
	}
	builder.WriteRune(r)
}
