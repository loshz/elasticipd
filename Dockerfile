FROM golang:1.16-alpine3.13
WORKDIR $GOPATH/src/github.com/syscll/elasticipd
COPY . .
RUN CGO_ENABLED=0 go install ./cmd/elasticipd

FROM alpine:3.13
COPY --from=0 /go/bin/elasticipd /bin/elasticipd
USER 2000:2000
CMD ["/bin/elasticipd"]
