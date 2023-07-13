FROM golang:alpine

ADD . /home

WORKDIR /home

CMD ["go","run","main.go"]