FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /veldoc ./cmd/veldoc

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /veldoc /app/veldoc

EXPOSE 8080
VOLUME ["/data"]
ENV VELDOC_ROOT=/data
ENV VELDOC_ADDR=:8080

ENTRYPOINT ["/app/veldoc"]
