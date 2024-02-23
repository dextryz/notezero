fmt:
	templ generate
	go mod tidy
	go fmt ./...

run:
	go run .
