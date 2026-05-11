FROM scratch

COPY ikca /app/ikca
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app

VOLUME ["/app/data"]

EXPOSE 20509

ENTRYPOINT ["/app/ikca"]
CMD ["-mode", "web", "-listen", ":20509", "-data-dir", "/app/data"]
