.PHONY: build build-hidden run clean

# Standard build
build:
	go build -o test ./cmd/test

# Windows: build without console window (fully hidden)
build-hidden:
	GOOS=windows go build -ldflags="-H windowsgui" -o test.exe ./cmd/test

# Run in background (Linux/macOS)
run: build
	./test &

clean:
	rm -f test test.exe
