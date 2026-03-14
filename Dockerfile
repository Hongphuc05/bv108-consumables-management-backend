FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server/main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata python3 py3-pip \
    && ln -sf /usr/bin/python3 /usr/bin/python

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY --from=builder /src/ubot-api /app/ubot-api

RUN python3 -m venv /opt/venv \
    && /opt/venv/bin/pip install --no-cache-dir -r /app/ubot-api/requirements.txt

EXPOSE 8080

ENV GIN_MODE=release
ENV PATH="/opt/venv/bin:${PATH}"
ENV PYTHON_PATH=/opt/venv/bin/python

CMD ["/app/server"]
