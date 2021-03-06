FROM golang:1.8

WORKDIR /src

RUN mkdir /config
ADD ./config /config/

ADD ./src /go/src/github.com/farice/EME/

RUN go get github.com/sirupsen/logrus && go get github.com/gomodule/redigo/redis && go get github.com/lib/pq

RUN chmod +x /config/entrypoint.sh
