FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.* ./
RUN env
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-s -w -static"' -o /deadman

# generate clean, final image for end users
FROM alpine:3.19

COPY --from=builder /deadman /deadman

USER 1000
EXPOSE 9095
ENTRYPOINT ["/deadman"]
