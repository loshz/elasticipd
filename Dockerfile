FROM golang:1.15-buster
WORKDIR $GOPATH/src/github.com/syscll/elasticipd
COPY . .
RUN go install

FROM alpine:3.12
COPY --from=0 /go/bin/elasticipd  /bin
USER 2000:2000
CMD ["/bin/elasticipd"]
