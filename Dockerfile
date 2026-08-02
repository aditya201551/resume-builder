# Build context is the repo root — this stage needs both backend/ and
# frontend/. Railway auto-detects this file since it's at the repo root.

# Build the frontend static bundle
FROM node:22-bookworm AS frontend-builder
WORKDIR /src
COPY package.json package-lock.json ./
COPY frontend/package.json frontend/package.json
COPY backend/package.json backend/package.json
RUN npm ci --workspace=frontend --include-workspace-root
COPY frontend frontend
RUN npm run build --workspace=frontend

# Build the Go binary
FROM golang:1.26-bookworm AS backend-builder
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend .
RUN CGO_ENABLED=0 go build -o /server ./cmd/api

# Runtime: chromedp/headless-shell already ships Chrome + its shared libs.
# We override its own entrypoint (which normally launches headless-shell as
# a long-running CDP server) with our Go binary — chromedp spawns the
# headless-shell binary itself as a short-lived child process per export
# request (see CHROME_EXEC_PATH), it doesn't need a standing CDP server.
FROM chromedp/headless-shell:stable
COPY --from=backend-builder /server /usr/local/bin/server
COPY --from=frontend-builder /src/frontend/dist /app/dist
ENV CHROME_EXEC_PATH=/headless-shell/headless-shell
ENV STATIC_DIR=/app/dist
ENTRYPOINT ["/usr/local/bin/server"]
