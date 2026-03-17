package pipeline_test

// Tests for the Run method can be added here.
// Run is the event loop that listens for hotkeys and processes captures.
// Testing Run is challenging due to goroutines and timing, but important for:
// - Context cancellation behavior
// - Goroutine cleanup
// - Queue management
// - Main event loop responsiveness
