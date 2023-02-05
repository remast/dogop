# 1. DogOp Builder
FROM golang as builder
WORKDIR /app
ADD . /app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o build/dogop .

# 2. DogOp Container
FROM alpine
COPY --from=builder /app/build/dogop /usr/bin/
EXPOSE 8080
ENTRYPOINT ["/usr/bin/dogop"]
