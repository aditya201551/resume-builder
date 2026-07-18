# Build the Go binary
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/api

# Runtime: chromedp/headless-shell already ships Chrome + its shared libs.
# We override its own entrypoint (which normally launches headless-shell as
# a long-running CDP server) with our Go binary — chromedp spawns the
# headless-shell binary itself as a short-lived child process per export
# request (see CHROME_EXEC_PATH), it doesn't need a standing CDP server.
FROM chromedp/headless-shell:stable
COPY --from=builder /server /usr/local/bin/server
ENV CHROME_EXEC_PATH=/headless-shell/headless-shell
ENTRYPOINT ["/usr/local/bin/server"]
