package ipc_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/vdyalex/lens-daemon/src/ipc"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

func TestWriteReadFrame_roundtrip(t *testing.T) {
	payload := []byte(`{"command":"status"}`)
	buffer := &bytes.Buffer{}

	if err := ipc.WriteFrame(buffer, payload); err != nil {
		t.Fatalf("WriteFrame() error: %v", err)
	}

	read, err := ipc.ReadFrame(buffer)
	if err != nil {
		t.Fatalf("ReadFrame() error: %v", err)
	}

	if !bytes.Equal(read, payload) {
		t.Errorf("Payload mismatch: expected %s, got %s", payload, read)
	}
}

func TestReadFrame_oversizedReturnsError(t *testing.T) {
	// Create a buffer with oversized length prefix
	buffer := &bytes.Buffer{}
	// Write 4-byte length: 5MB (exceeds 4MB limit)
	buffer.Write([]byte{0x00, 0x50, 0x00, 0x00}) // 5242880 in big-endian

	_, err := ipc.ReadFrame(buffer)
	if !errors.Is(err, exceptions.ErrIPCProtocolError) {
		t.Errorf("Expected ErrIPCProtocolError, got: %v", err)
	}
}

func TestReadFrame_truncatedReturnsError(t *testing.T) {
	// Write length but not enough payload data
	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0x00, 0x00, 0x00, 0x10}) // Length: 16 bytes
	buffer.Write([]byte("short"))                // Only 5 bytes

	_, err := ipc.ReadFrame(buffer)
	if !errors.Is(err, exceptions.ErrIPCProtocolError) {
		t.Errorf("Expected ErrIPCProtocolError, got: %v", err)
	}
}

func TestReadFrame_truncatedWrapsUnexpectedEOF(t *testing.T) {
	// Write length but not enough payload data
	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0x00, 0x00, 0x00, 0x10}) // Length: 16 bytes
	buffer.Write([]byte("short"))                // Only 5 bytes

	_, err := ipc.ReadFrame(buffer)

	// The error should wrap both ErrIPCProtocolError and io.ErrUnexpectedEOF
	if !errors.Is(err, exceptions.ErrIPCProtocolError) {
		t.Errorf("expected ErrIPCProtocolError in chain, got: %v", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("expected io.ErrUnexpectedEOF in chain, got: %v", err)
	}
}

func TestReadFrame_eofReturnsEof(t *testing.T) {
	buffer := &bytes.Buffer{} // Empty buffer

	_, err := ipc.ReadFrame(buffer)
	if !errors.Is(err, io.EOF) {
		t.Errorf("Expected io.EOF, got: %v", err)
	}
}
