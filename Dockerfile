FROM node:22-bookworm AS frontend-builder
WORKDIR /src
COPY package.json package-lock.json ./
COPY frontend/package.json frontend/package.json
COPY backend/package.json backend/package.json
RUN npm ci --workspace=frontend --include-workspace-root
COPY frontend frontend
RUN npm run build --workspace=frontend

FROM golang:1.26-bookworm AS backend-builder
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend .
RUN CGO_ENABLED=0 go build -o /server ./cmd/api

FROM chromedp/headless-shell:stable
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=backend-builder /server /usr/local/bin/server
COPY --from=frontend-builder /src/frontend/dist /app/dist
ENV CHROME_EXEC_PATH=/headless-shell/headless-shell
ENV STATIC_DIR=/app/dist
ENTRYPOINT ["/usr/local/bin/server"]
