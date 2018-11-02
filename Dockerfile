FROM golang
WORKDIR /go/src/github.com/digitorus/rfc2csv/
RUN go get -d -v golang.org/x/net/html  
COPY rfc2csv.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rfc2csv .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/digitorus/rfc2csv/rfc2csv .
CMD ["./rfc2csv"]  