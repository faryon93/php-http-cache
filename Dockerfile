# ----------------------------------------------------------------------------------------
# Image: Builder
# ----------------------------------------------------------------------------------------
FROM golang:alpine as builder

# setup the environment
ENV TZ=Europe/Berlin

# install dependencies
RUN apk --update --no-cache add git gcc musl-dev tzdata
WORKDIR /work
ADD ./ ./

# build the go binary
RUN go build -ldflags \
        '-X "main.BuildTime='$(date -Iminutes)'" \
         -X "main.GitCommit='$(git rev-parse --short HEAD)'" \
         -X "main.GitBranch='$(git rev-parse --abbrev-ref HEAD)'" \
         -s -w' \
         -v -o /tmp/http_cache .