FROM golang:1.19-alpine3.17 as builder

# Create appuser
RUN adduser -D -g '' appuser

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -extldflags '-static'" -o /app cmd/meldung/meldung.go
RUN chmod +x /app

FROM scratch

COPY --from=builder /app /app
COPY --from=builder /etc/passwd /etc/passwd

# Use an unprivileged user
USER appuser

WORKDIR /
CMD ["/app"]
