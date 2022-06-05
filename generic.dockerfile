FROM golang:1.18-alpine
WORKDIR /app

# Build and install decent-vcs-api
COPY . .
RUN go build

EXPOSE 8080
CMD ["./decent-vcs-api"]
