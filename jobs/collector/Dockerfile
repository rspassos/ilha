FROM golang:1.22.2-alpine

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace/jobs/collector

COPY jobs/collector/go.mod ./
RUN go mod download

COPY jobs/collector ./

CMD ["go", "run", "./cmd/collector", "--bootstrap-only"]
