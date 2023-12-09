FROM golang:1.21.5 as builder

WORKDIR /project

COPY go.* ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build -o bin/ptproc ./cmd/ptproc

FROM alpine:latest

WORKDIR /work

RUN apk --no-cache add ca-certificates

COPY --from=builder /project/bin/ptproc /bin/ptproc

ENTRYPOINT ["/bin/ptproc"]
