FROM golang:1
WORKDIR /usr/src/app
COPY . ./
RUN go build -o ackbar-example-client

ENTRYPOINT ["./ackbar-example-client"]
