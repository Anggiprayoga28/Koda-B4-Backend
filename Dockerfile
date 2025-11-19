FROM golang:alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN sed -i 's/go 1\.24\.0/go 1.24/' go.mod
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine
WORKDIR /app
COPY --from=build /app/main .
RUN mkdir uploads
EXPOSE 8080
CMD ["./main"]