FROM docker.io/golang:1.22.3@sha256:f43c6f049f04cbbaeb28f0aad3eea15274a7d0a7899a617d0037aec48d7ab010 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd

RUN go build -o ./build/main ./cmd/...

# ---

FROM gcr.io/distroless/base-debian12@sha256:cef75d12148305c54ef5769e6511a5ac3c820f39bf5c8a4fbfd5b76b4b8da843
WORKDIR /app

LABEL image.registry=ghcr.io
LABEL image.name=markormesher/ical-to-mqtt

COPY --from=builder /app/build/main /usr/local/bin/ical-to-mqtt

CMD ["/usr/local/bin/ical-to-mqtt"]
