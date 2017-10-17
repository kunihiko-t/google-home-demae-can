FROM golang:1.8-alpine

RUN apk update && \
  apk --no-cache add git mercurial curl make gcc g++ bash
