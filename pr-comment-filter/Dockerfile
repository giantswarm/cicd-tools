FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.19 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o pr-comment-filter main.go

FROM --platform=${TARGETPLATFORM:-linux/amd64} scratch
WORKDIR /app/
COPY --from=builder /app/pr-comment-filter /app/pr-comment-filter
ENTRYPOINT ["/app/pr-comment-filter"]
