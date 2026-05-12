FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ikca .

FROM scratch
COPY --from=builder /build/ikca /app/ikca
COPY --from=builder /build/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
VOLUME ["/app/data"]
EXPOSE 20509
ENTRYPOINT ["/app/ikca"]
CMD ["-mode", "web", "-listen", ":20509", "-data-dir", "/app/data"]
