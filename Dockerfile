FROM golang:1.9 as builder

RUN mkdir /app
ADD . /app/
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main -v

FROM scratch
COPY --from=builder /app/main /
CMD ["/main"]
