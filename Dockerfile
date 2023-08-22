FROM golang:1.20-alpine AS build
WORKDIR /go/src/simple-log-storage
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/simpleLogStorage ./cmd/simpleLogStorage

FROM scratch
COPY --from=build /go/bin/simpleLogStorage /bin/simpleLogStorage

ENTRYPOINT ["/bin/simple-log-storage"]
