############################
# 1. Build stage           #
############################
FROM golang:1.22-alpine AS builder

WORKDIR /src

# cache deps first
COPY go.mod ./

# copy source
# TODO copy only
COPY . .

# build main binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=integration -ldflags "-s -w" -o /queue-broker ./cmd/server

############################
# 2. Runtime stage         #
############################
FROM scratch

COPY --from=builder /queue-broker /

EXPOSE 8080

ENTRYPOINT ["/queue-broker"]