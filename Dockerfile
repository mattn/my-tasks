FROM golang:1.9-alpine as base
WORKDIR /usr/src
COPY . .
RUN apk add git
RUN CGO_ENABLED=0 go get -d -ldflags "-s -w" .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o main

FROM scratch
COPY --from=base /usr/src/main /go-http-microservice
CMD ["/go-http-microservice"]
