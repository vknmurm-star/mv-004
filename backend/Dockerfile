# syntax=docker/dockerfile:1.6

# ---- Build stage ----
FROM golang:1.22-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build each binary
RUN go build -trimpath -ldflags="-s -w" -o /out/server  ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server  /app/server
COPY --from=build /out/migrate /app/migrate
COPY internal/migrations        /app/migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]