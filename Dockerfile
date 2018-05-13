FROM golang:alpine
LABEL maintainer="danbondd@gmail.com"
RUN mkdir -p /go/src/github.com/danbondd/elasticipd
COPY . /go/src/github.com/elasticipd
RUN go install

FROM alpine
# Manually add dependencies
RUN apk add --no-cache ca-certificates
RUN apk add --no-cache python3
# Install awscli
RUN pip3 install --upgrade pip && \
    pip3 install awscli
COPY --from=0 /go/bin/elasticipd  /usr/local/bin