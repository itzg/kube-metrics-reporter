FROM golang:1.13 as builder

WORKDIR /build
COPY go.mod go.sum /build/
RUN go mod download
COPY . /build

# gcflags described by IntelliJ's remote Go debug info
RUN CGO_ENABLED=0 go build -gcflags "all=-N -l"

# skaffold Go debugging doesn't work with alpine or scratch
FROM ubuntu
COPY --from=builder /build/kube-metrics-reporter /usr/bin
# identify as Go for skaffold debug
ENV GOTRACEBACK=all
ENTRYPOINT ["/usr/bin/kube-metrics-reporter"]