FROM golang:1.12
RUN mkdir /tmp/build
WORKDIR /tmp/build
COPY lotto.go .
COPY go.mod .
RUN CGO_ENABLED=0 go build -o /lotto lotto.go

FROM scratch
EXPOSE 8080
COPY --from=0 /lotto /lotto
CMD ["/lotto"]
