FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ikca .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/ikca .
COPY --from=builder /build/data ./data

VOLUME ["/app/data"]

EXPOSE 20509

ENTRYPOINT ["./ikca"]
CMD ["-mode", "web", "-listen", ":20509", "-data-dir", "/app/data"]
