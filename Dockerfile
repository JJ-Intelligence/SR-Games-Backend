# Pull Go image
FROM golang:1.16-alpine as builder
ENV GO111MODULE=on

# Set working directory
WORKDIR /sr-games-backend

# Cache go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy backend to container image
COPY . .

# Build binary (backend.exe) inside container
RUN CGO_ENABLED=0 GOOS=linux go build -o backend.exe cmd/main.go

# Create production image
FROM scratch
COPY --from=builder /sr-games-backend/backend.exe /backend.exe
ENV PORT 8081
EXPOSE 8081
CMD ["./backend.exe", "8081"]

#FROM alpine
#RUN apk add --no-cache ca-certificates
#
## Copy binary to production image
#COPY --from=builder /backend.exe /
#
## Set environment variables
#ENV PORT 8080
#
## Run project
#CMD ["/backend"]
