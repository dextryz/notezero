# Define binary output name
BINARY_NAME=tenet

# run: Runs the Go application
run:
	go run ./cmd/main.go

# build: Builds the Go application binary
build:
	go build -o $(BINARY_NAME) ./cmd/main.go

# clean: Cleans up the binary
clean:
	go clean
	rm -f $(BINARY_NAME)

fmt:
	templ generate
	go mod tidy
	go fmt ./...
