FROM golang:latest as build
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./...
RUN make build-dist

FROM scratch
COPY --from=build /go/src/app/bin/summit /app
ENTRYPOINT ["/app"]
EXPOSE 5000 5002