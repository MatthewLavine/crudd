FROM golang:1.18

WORKDIR $GOPATH/src/crudd

COPY . .

RUN go build -v -o $GOPATH/bin/crudd .

EXPOSE 4901

CMD ["crudd"]
