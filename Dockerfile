FROM golang:1
WORKDIR /usr/src/app
COPY . ./
RUN go build -o ackbar

EXPOSE 8080
ENTRYPOINT ["./ackbar"]
