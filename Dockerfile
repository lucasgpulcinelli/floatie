FROM golang:1.22.3-alpine

WORKDIR /srv

COPY ./go.mod ./go.sum /srv/
RUN go mod download

COPY . /srv/
RUN go build -o main -v github.com/lucasgpulcinelli/floatie/cmd/main

FROM alpine:latest

WORKDIR /srv

COPY --from=0 /srv/main /srv/main

EXPOSE 8080
EXPOSE 9999

ENTRYPOINT ["/srv/main"]
