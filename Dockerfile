# Pull Ubuntu image and install Go & GCC (required to use CGO which we need for plugins)
FROM alpine as builder
RUN apk update
RUN apk upgrade
RUN apk add --update go=1.16.7-r0 gcc=10.3.1_git20210424-r2 g++=10.3.1_git20210424-r2

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
RUN go build -buildmode=plugin -o tictactoe.so plugins/games/tictactoe/main.go


# Create production image
FROM alpine
COPY --from=builder /sr-games-backend/backend.exe .
COPY --from=builder /sr-games-backend/config.yaml .
COPY --from=builder /sr-games-backend/tictactoe.so /plugins/games/

ENV FRONTEND_HOST "https://sr-games.herokuapp.com"
ENV CONFIG_PATH "./config.yaml"

ENV PORT 80
EXPOSE 80
ENTRYPOINT ["./backend.exe"]
