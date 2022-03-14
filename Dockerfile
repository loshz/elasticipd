#################
# Build stage 0 #
#################
FROM golang:1.17-alpine3.15

# Create work dir
WORKDIR $GOPATH/src/github.com/loshz/elasticipd

# Copy required files
COPY ./cmd/elasticipd .
COPY go.mod .
COPY go.sum .

# Build binary
ARG BUILD_NUMBER
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install \
	--ldflags="-X github.com/loshz/elasticipd/main.version=$BUILD_NUMBER" .

#################
# Build stage 1 #
#################
FROM alpine:3.15
COPY --from=0 /go/bin/elasticipd /bin/elasticipd
USER 2000:2000
CMD ["/bin/elasticipd"]
