# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS frontend-builder
WORKDIR /src
COPY package.json package-lock.json ./
COPY frontend/package.json frontend/package.json
COPY backend/package.json backend/package.json
# Railway requires literal, service-scoped cache IDs (build args cannot be used).
RUN --mount=type=cache,id=s/a6496064-610c-4d15-94c7-294f595f1575-/root/.npm,target=/root/.npm \
    npm ci --workspace=frontend --include-workspace-root=false --no-audit --no-fund
# Catalog changes should rebuild the frontend without reinstalling dependencies.
COPY backend/internal/design/templates.json backend/internal/design/templates.json
COPY frontend frontend
RUN npm run build --workspace=frontend

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS backend-builder
WORKDIR /src/backend
ENV CGO_ENABLED=0 GOWORK=off
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,id=s/a6496064-610c-4d15-94c7-294f595f1575-/go/pkg/mod,target=/go/pkg/mod \
    go mod download
COPY backend/cmd ./cmd
COPY backend/internal ./internal
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,id=s/a6496064-610c-4d15-94c7-294f595f1575-/go/pkg/mod,target=/go/pkg/mod \
    --mount=type=cache,id=s/a6496064-610c-4d15-94c7-294f595f1575-/root/.cache/go-build,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -buildvcs=false -ldflags="-s -w" -o /server ./cmd/api

# PDF preview/download run in the client. The legacy Chrome export endpoint is
# not supported by this image. Distroless includes TLS roots and timezone data.
FROM gcr.io/distroless/static-debian13:nonroot AS runtime
WORKDIR /app
COPY --from=backend-builder /server /usr/local/bin/server
COPY --from=frontend-builder /src/frontend/dist /app/dist
ENV STATIC_DIR=/app/dist
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
