ARG ALPINEVERSION="3.20"
ARG GOVERSION="1.23"

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

WORKDIR /

COPY --from=builder /app /app
COPY config /config

EXPOSE 8080
HEALTHCHECK CMD curl --fail http://localhost:8080 || exit 1

CMD ["/app"]
