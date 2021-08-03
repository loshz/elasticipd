#################
# Build stage 0 #
#################
FROM golang:1.16-alpine3.14

# Create work dir
WORKDIR $GOPATH/src/github.com/syscll/elasticipd

# Copy required files
COPY ./cmd/elasticipd .
COPY go.mod .
COPY go.sum .

# Build binary
ARG BUILD_NUMBER
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install \
	--ldflags="-X github.com/syscll/elasticipd/main.version=$BUILD_NUMBER" .

#################
# Build stage 1 #
#################
FROM alpine:3.14
COPY --from=0 /go/bin/elasticipd /bin/elasticipd
USER 2000:2000
CMD ["/bin/elasticipd"]
