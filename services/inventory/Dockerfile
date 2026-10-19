# syntax=docker/dockerfile:1

# --- Build stage: full Go toolchain ---
FROM golang:1.26 AS build
WORKDIR /src

# Copy dependency manifests first so this layer is cached until they change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/inventory .

# --- Runtime stage: no shell, no package manager, just the binary ---
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/inventory /inventory
EXPOSE 50051
USER nonroot:nonroot
ENTRYPOINT ["/inventory"]