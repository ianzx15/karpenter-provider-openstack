
FROM golang:1.25.3-alpine AS builder

ARG MODULE=github.com/ianzx15/karpenter-provider-openstack
ENV CGO_ENABLED=0

WORKDIR /go/src/${MODULE}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags "-s -w" -o /app/controller ./cmd/controller/main.go

FROM gcr.io/distroless/static-debian11

ENTRYPOINT ["/controller"]

USER 65532:65532

COPY --from=builder /app/controller /controller
