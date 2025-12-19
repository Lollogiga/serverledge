# Build
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /native-executor main.go

# Runtime
FROM alpine:3.20
WORKDIR /app
COPY --from=build /native-executor /native-executor
EXPOSE 8080
ENTRYPOINT ["/native-executor"]