package ipc

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/vdyalex/lens-daemon/src/utils/constants"
	"github.com/vdyalex/lens-daemon/src/utils/exceptions"
)

// WriteFrame writes buf as a length-prefixed frame to w.
// The 4-byte big-endian length field precedes the payload.
// Returns ErrIPCProtocolError if len(buf) exceeds IPCMaxFrameSize.
func WriteFrame(w io.Writer, buf []byte) error {
	if len(buf) > constants.IPCMaxFrameSize {
		return fmt.Errorf("%w: frame size %d exceeds max %d", exceptions.ErrIPCProtocolError, len(buf), constants.IPCMaxFrameSize)
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(buf)))

	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}

// ReadFrame reads one length-prefixed frame from r.
// Blocks until a complete frame arrives or r returns an error.
// Returns ErrIPCProtocolError if the declared length exceeds maxFrameSize
// (guards against malformed or malicious length fields).
func ReadFrame(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, fmt.Errorf("%w: failed to read length prefix: %v", exceptions.ErrIPCProtocolError, err)
	}

	frameLen := binary.BigEndian.Uint32(lenBuf)
	if frameLen > uint32(constants.IPCMaxFrameSize) {
		return nil, fmt.Errorf("%w: declared frame size %d exceeds max %d", exceptions.ErrIPCProtocolError, frameLen, constants.IPCMaxFrameSize)
	}

	buf := make([]byte, frameLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("%w: failed to read frame payload: %v", exceptions.ErrIPCProtocolError, err)
	}
	return buf, nil
}
