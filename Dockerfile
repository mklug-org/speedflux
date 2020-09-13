FROM golang:1.15 as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go get -d
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine
#FROM scratch
RUN adduser -S -D -H -h /app appuser
USER appuser
#USER 10001
COPY --from=builder /build/main /app/
WORKDIR /app
CMD ["./main"]