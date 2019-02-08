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

# ----------------------------------------------------------------------------------------
# Image: Deployment
# ----------------------------------------------------------------------------------------
FROM alpine:latest
MAINTAINER Maximilian Pachl <m@ximilian.info>

RUN apk --update --no-cache add ca-certificates tzdata bash su-exec curl

# add relevant files to container
COPY --from=builder /tmp/http_cache /usr/sbin/http_cache

# make binary executable
RUN chown nobody:nobody /usr/sbin/http_cache && \
    chmod +x /usr/sbin/http_cache

EXPOSE 8000
CMD /usr/sbin/http_cache
