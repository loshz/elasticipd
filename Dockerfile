FROM golang:1.12-alpine
WORKDIR /go/src/github.com/syscll/elasticipd
COPY . .
RUN go install

FROM alpine
# Manually add dependencies
RUN apk add --no-cache ca-certificates
# Copy binary from build container
COPY --from=0 /go/bin/elasticipd  /usr/local/bin
