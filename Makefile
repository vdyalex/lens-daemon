.PHONY: build run clean

build:
	go build -o test ./cmd/test

run: build
	./test &

clean:
	rm -f test
