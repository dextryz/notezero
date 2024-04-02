# Define binary output name
SERVER_NAME=server
CLI_NAME=nz

# run: Runs the Go application
run:
	go run ./cmd/server/main.go

# build: Builds the Go application binary
build-server:
	go build -o $(BINARY_NAME) ./cmd/server/main.go

build-cli:
	go build -o $(CLI_NAME) ./cmd/nz/main.go

# clean: Cleans up the binary
clean:
	go clean
	rm -f $(BINARY_NAME)

fmt:
	templ generate
	go mod tidy
	go fmt ./...
