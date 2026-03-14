.PHONY: build run clean vet fmt check

build:
	@LEPTONICA_VER=$$(ls $$(brew --cellar leptonica) | tail -1) && \
	TESSERACT_VER=$$(ls $$(brew --cellar tesseract) | tail -1) && \
	CGO_CFLAGS="-I$$(brew --cellar leptonica)/$$LEPTONICA_VER/include -I$$(brew --cellar tesseract)/$$TESSERACT_VER/include" \
	CGO_CXXFLAGS="-I$$(brew --cellar leptonica)/$$LEPTONICA_VER/include -I$$(brew --cellar tesseract)/$$TESSERACT_VER/include" \
	CGO_LDFLAGS="-L$$(brew --cellar leptonica)/$$LEPTONICA_VER/lib -L$$(brew --cellar tesseract)/$$TESSERACT_VER/lib" \
	go build -o networkd ./src

run: build
	./networkd &

clean:
	rm -f networkd

vet:
	go vet ./...

fmt:
	gofmt -w .

check: fmt vet
