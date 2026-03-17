package ipc

import (
	"strings"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// NewLogBroker creates a new LogBroker ready to use.
func NewLogBroker() *LogBroker {
	return &LogBroker{
		subscribers: make(map[int]chan LogEvent),
	}
}

// Subscribe registers a new subscriber channel. Returns subscriber ID and channel.
// The caller owns the channel and must call Unsubscribe when done.
func (b *LogBroker) Subscribe() (int, <-chan LogEvent) {
	ch := make(chan LogEvent, constants.IPCLogSubscriberBuffer)

	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	b.subscribers[id] = ch

	return id, ch
}

// Unsubscribe removes a subscriber by ID and closes its channel.
func (b *LogBroker) Unsubscribe(id int) {
	b.mu.Lock()
	ch, ok := b.subscribers[id]
	if ok {
		delete(b.subscribers, id)
	}
	b.mu.Unlock()

	if ok {
		close(ch)
	}
}

// Write implements io.Writer. Distributes p to all active subscribers without blocking.
// Slow subscribers that cannot receive immediately are dropped (channel full is a no-op).
// Parses slog key-value lines like: time=... level=INFO msg=... key=val ...
// Returns number of bytes written (always len(p)) or error.
func (b *LogBroker) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	event := parseSlogLine(p)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- event:
			// Sent successfully
		default:
			// Channel full; drop subscriber silently (non-blocking)
		}
	}

	return len(p), nil
}

// Count returns the current number of active subscribers.
func (b *LogBroker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// parseSlogLine parses a slog text line into a LogEvent.
// Format: time=2026-03-17T... level=INFO msg="message" key=value ...
func parseSlogLine(p []byte) LogEvent {
	line := strings.TrimSpace(string(p))

	event := LogEvent{
		Time:  time.Now(),
		Level: "INFO",
		Attrs: make(map[string]any),
	}

	// Parse key=value pairs
	// slog format: time=2026-03-17T12:34:56... level=DEBUG msg="text with spaces" key=value
	i := 0
	for i < len(line) {
		// Skip whitespace
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}

		// Find next '='
		keyStart := i
		for i < len(line) && line[i] != '=' {
			i++
		}
		if i >= len(line) {
			break
		}

		key := line[keyStart:i]
		i++ // skip '='

		var value string
		// Check if value is quoted
		if i < len(line) && line[i] == '"' {
			i++ // skip opening quote
			var sb strings.Builder
			for i < len(line) {
				if line[i] == '\\' && i+1 < len(line) {
					// Handle escaped characters
					i++
					switch line[i] {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					case '\\':
						sb.WriteByte('\\')
					case '"':
						sb.WriteByte('"')
					default:
						// Unknown escape; write as-is
						sb.WriteByte('\\')
						sb.WriteByte(line[i])
					}
					i++
				} else if line[i] == '"' {
					i++ // skip closing quote
					break
				} else {
					sb.WriteByte(line[i])
					i++
				}
			}
			value = sb.String()
		} else {
			// Unquoted value; read until space
			valStart := i
			for i < len(line) && line[i] != ' ' {
				i++
			}
			value = line[valStart:i]
		}

		// Store parsed fields
		switch key {
		case "time":
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				event.Time = t
			}
		case "level":
			event.Level = value
		case "msg":
			event.Message = value
		default:
			event.Attrs[key] = value
		}
	}

	// Fallback if msg was not found
	if event.Message == "" {
		event.Message = strings.TrimSpace(line)
	}

	return event
}
