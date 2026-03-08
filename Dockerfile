FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o water-research .

FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app
RUN mkdir -p static/proposals

COPY --from=builder /app/water-research .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./water-research"]
