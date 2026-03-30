package ipc

import (
	"strings"
	"sync"
	"time"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

// LogBroker is a fan-out io.Writer that distributes log lines to registered subscribers.
// It parses slog text lines into LogEvent structs, stores them in a ring buffer for
// late-subscriber replay, and sends them to all subscribers.
// It is safe for concurrent use. Slow subscribers are dropped (non-blocking send).
type LogBroker struct {
	mu          sync.RWMutex
	subscribers map[int]chan LogEvent
	nextID      int
	ring        []LogEvent // circular replay buffer, pre-allocated to IPCLogRingBuffer
	ringHead    int        // index of the oldest valid event
	ringLength  int        // number of valid events stored (0..len(ring))
}

// NewLogBroker creates a new LogBroker ready to use.
func NewLogBroker() *LogBroker {
	return &LogBroker{
		subscribers: make(map[int]chan LogEvent),
		ring:        make([]LogEvent, constants.IPCLogRingBuffer),
	}
}

// Subscribe registers a new subscriber channel. Returns subscriber ID and channel.
// The ring buffer is replayed into the channel (oldest to newest) before the subscriber
// is added to the active map, so the caller receives recent history without gaps.
// The caller owns the channel and must call Unsubscribe when done.
func (b *LogBroker) Subscribe() (int, <-chan LogEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Size the channel to hold the full replay plus live headroom.
	ch := make(chan LogEvent, b.ringLength+constants.IPCLogSubscriberBuffer)

	// Replay ring buffer (oldest → newest) before registering so no events are missed.
	for i := range b.ringLength {
		ch <- b.ring[(b.ringHead+i)%len(b.ring)]
	}

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

// Write implements io.Writer. Parses slog text lines into LogEvent structs,
// stores them in the ring buffer, and distributes to all active subscribers.
// Slow subscribers that cannot receive immediately are dropped (non-blocking send).
// Returns number of bytes written (always len(p)) or error.
func (b *LogBroker) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	event := parseSlogLine(p)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Store event in ring buffer; overwrite oldest entry when full.
	index := (b.ringHead + b.ringLength) % len(b.ring)
	b.ring[index] = event
	if b.ringLength < len(b.ring) {
		b.ringLength++
	} else {
		b.ringHead = (b.ringHead + 1) % len(b.ring)
	}

	// Fan out to active subscribers (non-blocking; slow subscribers are dropped).
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
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
		case constants.SlogFieldTime:
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				event.Time = t
			}
		case constants.SlogFieldLevel:
			event.Level = value
		case constants.SlogFieldMessage:
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
