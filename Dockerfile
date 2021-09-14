#build stage
FROM golang:alpine AS builder
RUN apk add --no-cache git

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -o /go/bin/app -v ./...

#Get swagger
FROM alpine:latest as swagger
ADD https://github.com/swagger-api/swagger-ui/archive/refs/tags/v3.52.1.tar.gz .
RUN tar -xzf v3.52.1.tar.gz
RUN mv swagger-ui-3.52.1/dist dist
RUN sed -i 's+https://petstore.swagger.io/v2/swagger.json+shorty.yaml+g' dist/index.html
COPY api/shorty.yaml dist

#final stage
FROM alpine:latest
COPY --from=builder /go/bin/app /app
COPY --from=swagger dist swagger-dist
ENTRYPOINT /app
LABEL Name=shorty Version=0.0.1
ENV PORT 8080
EXPOSE ${PORT}
