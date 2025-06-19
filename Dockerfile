FROM golang:1.24

WORKDIR $GOPATH/src/crudd

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o $GOPATH/bin/crudd .

EXPOSE 4901

CMD ["crudd"]
