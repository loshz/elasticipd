FROM golang:alpine
WORKDIR /go/src/github.com/syscll/elasticipd
COPY . .
RUN go install

FROM alpine
# Manually add dependencies
RUN apk add --no-cache ca-certificates
RUN apk add --no-cache python3
# Install awscli
RUN pip3 install --upgrade pip && \
    pip3 install awscli
# Copy binary from build container
COPY --from=0 /go/bin/elasticipd  /usr/local/bin
