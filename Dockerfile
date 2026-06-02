# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/swedish ./

# Run as root: Fly's mounted volumes are root-owned and distroless has no
# shell to chown them in an entrypoint. Distroless still gives us a minimal
# attack surface (no shell, no package manager, ~2MB base).
FROM gcr.io/distroless/static
COPY --from=build /out/swedish /swedish
EXPOSE 8080
ENTRYPOINT ["/swedish"]
