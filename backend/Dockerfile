FROM golang:1.25-alpine AS builder
WORKDIR /src
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${TARGET}

FROM alpine:3.22
ARG TARGET=api
RUN apk add --no-cache ca-certificates tzdata \
    && if [ "$TARGET" = "worker" ]; then apk add --no-cache poppler-utils tesseract-ocr tesseract-ocr-data-chi_sim tesseract-ocr-data-eng; fi \
    && adduser -D -u 10001 app
USER app
COPY --from=builder /out/app /app
ENTRYPOINT ["/app"]
