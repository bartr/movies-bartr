# syntax=docker/dockerfile:1.7

# Build stage --------------------------------------------------------------
FROM golang:1.26-bookworm AS build

WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=0.3.0
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN go build \
      -trimpath \
      -ldflags "-s -w -X github.com/bartr/bartr-movies/internal/version.Version=${VERSION}" \
      -o /out/movies-api \
      ./cmd/movies-api

# Runtime stage ------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/movies-api /movies-api

# Bake the JSON catalog into the image at /data per spec §5.2. Loader
# runs in main.go after startup; /readyz gates on it.
COPY data /data

# Spec §8.1 wants uid 1000; the Pod sets runAsUser explicitly, but we also
# default to a non-root uid here so `docker run` is non-root by default.
USER 1000:1000

EXPOSE 8080

ENTRYPOINT ["/movies-api"]
