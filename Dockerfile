FROM golang:1.11.2-alpine3.8 as builder

RUN apk add --no-cache --virtual .build-deps \
    alpine-sdk \
    cmake \
    sudo \
    libssh2 libssh2-dev\
    git \
    xz

WORKDIR /go/src/app

ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .

RUN go mod download

ADD . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main main.go

# strip and compress the binary
RUN strip --strip-unneeded main

# use a minimal alpine image
FROM alpine:3.8
# add ca-certificates in case you need them
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
# set working directory
WORKDIR /root
# copy the binary from builder
COPY --from=builder /go/src/app/main .

# run the binary
CMD ["./main"]
