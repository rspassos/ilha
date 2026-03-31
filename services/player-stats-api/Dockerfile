FROM golang:1.22.2-alpine

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace/services/player-stats-api

COPY services/player-stats-api/go.mod ./
COPY services/player-stats-api/go.sum ./
RUN go mod download

COPY services/player-stats-api ./

CMD ["go", "run", "./cmd/player-stats-api"]
