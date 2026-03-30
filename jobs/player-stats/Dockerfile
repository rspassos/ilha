FROM golang:1.22.2-alpine

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace/jobs/player-stats

COPY jobs/player-stats/go.mod ./
RUN go mod download

COPY jobs/player-stats ./

CMD ["go", "run", "./cmd/player-stats", "--bootstrap-only"]
