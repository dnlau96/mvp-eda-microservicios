FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod ./
COPY main.go .
RUN go mod tidy && go build -o /panel-service main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=build /panel-service /panel-service
CMD ["/panel-service"]
