FROM golang:1 as build
WORKDIR /usr/src/app
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o ackbar

FROM scratch

COPY --from=build /usr/src/app/ackbar /bin/ackbar

EXPOSE 8080
CMD ["/bin/ackbar"]
