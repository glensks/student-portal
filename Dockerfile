FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o student-portal .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/student-portal .
COPY --from=builder /app/frontend ./frontend
RUN mkdir -p uploads
EXPOSE 8080
CMD ["./student-portal"]
