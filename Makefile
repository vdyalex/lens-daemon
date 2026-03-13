.PHONY: build build-hidden run clean

# Standard build
build:
	go build -o ccat ./cmd/ccat

# Windows: build without console window (fully hidden)
build-hidden:
	GOOS=windows go build -ldflags="-H windowsgui" -o ccat.exe ./cmd/ccat

# Run in background (Linux/macOS)
run: build
	./ccat &

clean:
	rm -f ccat ccat.exe
