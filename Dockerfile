ARG ALPINEVERSION="3.23"
ARG GOVERSION="1.25"

FROM node:current-alpine${ALPINEVERSION} as web

WORKDIR /app
COPY internal/web .
RUN npm install && npm run build

FROM golang:${GOVERSION}-alpine${ALPINEVERSION} AS dependencies

ENV GOPATH="/go"

RUN mkdir -p "$GOPATH/src" "$GOPATH/pkg"

WORKDIR /src
COPY go.mod go.sum ./

RUN go mod download

FROM golang:${GOVERSION}-alpine${ALPINEVERSION} AS builder

ENV GOPATH="/go"

COPY --from=dependencies $GOPATH/pkg $GOPATH/pkg
COPY --from=dependencies $GOPATH/src $GOPATH/src

WORKDIR /src

COPY . .
COPY --from=web /app/dist internal/web/dist

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app cmd/meldung/meldung.go

FROM alpine:${ALPINEVERSION}

# ca-certificates is required for making HTTPS requests to services like matrix, rocketchat, etc.
# curl is required for healthchecks.
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /

COPY --from=builder /app /app
COPY config /config

# Create writable directories for runtime data
RUN mkdir -p /files && chown appuser:appgroup /files

USER appuser

EXPOSE 8080
HEALTHCHECK CMD curl --fail http://localhost:8080 || exit 1

CMD ["/app"]
