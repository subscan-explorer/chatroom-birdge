FROM golang:1.24 as build

ENV GOOS linux
ENV GOARCH=amd64

RUN apt-get update && apt-get install -y \
    libolm-dev \
    libsqlite3-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/* \

WORKDIR /build/cache
ADD go.mod .
ADD go.sum .
RUN go mod download

WORKDIR /workspace/release

ADD . .
RUN go build -o chatroom cmd/main.go

FROM alpine as prod

COPY --from=build /workspace/release/chatroom /chatroom/bin/chatroom

WORKDIR /workspace/

CMD ["/chatroom/bin/chatroom"]



