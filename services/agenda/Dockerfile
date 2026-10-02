FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /agenda ./services/agenda/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /agenda /agenda
CMD ["/agenda"]
