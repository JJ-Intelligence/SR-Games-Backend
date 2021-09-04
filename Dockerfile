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

# Build TicTacToe plugin inside container
RUN CGO_ENABLED=0 GOOS=linux go build \
    -buildmode=plugin \
    -o tictactoe.so \
    plugins/games/tictactoe/main.go

# Create production image
FROM scratch
COPY --from=builder /sr-games-backend/backend.exe /backend.exe
COPY --from=builder /sr-games-backend/config.yaml /config.yaml
COPY --from=builder /sr-games-backend/tictactoe.so /plugins/games/tictactoe.so

ENV FRONTEND_HOST https://sr-games.herokuapp.com
ENV CONFIG_PATH /config.yaml

ENV PORT 80
EXPOSE 80
CMD ["./backend.exe"]
