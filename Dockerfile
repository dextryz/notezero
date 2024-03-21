# --------------------------------------------------------------------

# Go build stage
FROM golang:1.22-alpine3.19 as builder

RUN apk add --no-cache gcc musl-dev sqlite-dev ca-certificates

# Set a temporary work directory
WORKDIR /app

# Add necessary go files
COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Generate Go codes from template files
RUN go run -mod=mod github.com/a-h/templ/cmd/templ@latest generate

# Build the go binary
#RUN CGO_ENABLED=1 GOOS=linux go build -o run -a -ldflags '-linkmode external -extldflags "-static"' ./cmd/server/main.go
#RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-s -w -extldflags "-static"'-o server ./cmd/server/main.go
RUN CGO_ENABLED=1 GOOS=linux go build -tags 'osusergo netgo static_build' -ldflags '-extldflags "-static"' -o server ./cmd/server/main.go

# --------------------------------------------------------------------
# Build final image
FROM scratch

# Copy Go binary
COPY --from=builder /app/fonts ./fonts
COPY --from=builder /app/static ./static
COPY --from=builder /app/server .

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

# Run the application
CMD ["/server"]

# --------------------------------------------------------------------
