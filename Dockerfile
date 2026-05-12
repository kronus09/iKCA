FROM alpine:3.20 AS ca-certs
RUN apk add --no-cache ca-certificates

FROM scratch
COPY ikca /app/ikca
COPY --from=ca-certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
VOLUME ["/app/data"]
EXPOSE 20509
ENTRYPOINT ["/app/ikca"]
CMD ["-mode", "web", "-listen", ":20509", "-data-dir", "/app/data"]
