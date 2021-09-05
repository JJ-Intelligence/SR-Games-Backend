# Pull Go image
FROM golang:1.16-buster as builder
ENV GO111MODULE on

# Set working directory
WORKDIR /sr-games-backend

# Cache go modules
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy backend to container image
COPY . .

# Must use cgo/linux for plugins
ENV CGO_ENABLED 1
ENV GOOS linux

# Build binary (backend.exe) inside container
RUN go build -o backend.exe cmd/main.go

# Build TicTacToe plugin inside container
RUN GOOS=linux go build -buildmode=plugin -o tictactoe.so plugins/games/tictactoe/main.go


# Create production image
FROM alpine
COPY --from=builder /sr-games-backend/backend.exe /
COPY --from=builder /sr-games-backend/config.yaml /
COPY --from=builder /sr-games-backend/tictactoe.so /plugins/games/tictactoe.so

ENV FRONTEND_HOST "https://sr-games.herokuapp.com"
ENV CONFIG_PATH "./config.yaml"

RUN ls
RUN ls plugins/games

ENV PORT 80
EXPOSE 80
ENTRYPOINT ["./backend.exe"]
