# --------------------------------------------------------------------

# Go build stage
FROM golang:1.22 as gobuilder

# Set a temporary work directory
WORKDIR /app

# Add necessary go files
COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Generate Go codes from template files
RUN go run -mod=mod github.com/a-h/templ/cmd/templ@latest generate

# Build the go binary
RUN CGO_ENABLED=1 GOOS=linux go build -o run -a -ldflags '-linkmode external -extldflags "-static"' ./cmd/server/main.go

# --------------------------------------------------------------------

# Build final image
FROM scratch

# Copy Go binary
COPY --from=gobuilder /app/run .

EXPOSE 8080

# Run the application
CMD ["/run"]

# --------------------------------------------------------------------
