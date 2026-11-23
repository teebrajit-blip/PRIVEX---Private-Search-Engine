FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN go mod tidy && go build -o /privex ./cmd/privex-server

FROM gcr.io/distroless/base-debian12
COPY --from=build /privex /privex
COPY --from=build /src/testdata/corpus.json /testdata/corpus.json
COPY --from=build /src/web /web
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/privex"]
