.PHONY: build run clean

build:
	go build -o ccat ./cmd/ccat

run: build
	./ccat &

clean:
	rm -f ccat
