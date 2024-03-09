run:
	@go run cmd/pingg/main.go 8.8.8.8

build:
	@go build -ldflags='-s -w' -o bin/pingg.exe ./cmd/pingg/
install:
	@go install ./cmd/pingg/