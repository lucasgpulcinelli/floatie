FROM golang:1.21.8-alpine

WORKDIR /srv

COPY ./go.mod ./go.sum /srv/
RUN go mod download

COPY . /srv/
RUN go build -o main -v github.com/lucasgpulcinelli/floatie/cmd/main

FROM alpine:latest
EXPOSE 8080

COPY --from=0 /srv/main /srv/main

ENTRYPOINT ["/srv/main"]
