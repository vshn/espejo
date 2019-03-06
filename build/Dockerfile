FROM docker.io/library/golang:1.11 as builder

WORKDIR /go/src/github.com/vshn/espejo

COPY ./ ./

RUN go test \
         -race \
         -failfast \
         -v ./...

RUN CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix nocgo -ldflags '-extldflags "-static"' \
      -o espejo cmd/manager/main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/vshn/espejo/espejo /bin/

USER 1001

ENTRYPOINT ["/bin/espejo"]
