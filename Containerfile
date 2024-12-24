FROM docker.io/golang:1.22.3@sha256:f43c6f049f04cbbaeb28f0aad3eea15274a7d0a7899a617d0037aec48d7ab010 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd

RUN go build -o ./build/main ./cmd/...

# ---

FROM gcr.io/distroless/base-debian12@sha256:1aae189e3baecbb4044c648d356ddb75025b2ba8d14cdc9c2a19ba784c90bfb9
WORKDIR /app

LABEL image.registry=ghcr.io
LABEL image.name=markormesher/ical-to-mqtt

COPY --from=builder /app/build/main /usr/local/bin/ical-to-mqtt

CMD ["/usr/local/bin/ical-to-mqtt"]
