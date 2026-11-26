FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /wichteln .

FROM scratch
COPY --from=build /wichteln /wichteln
ENV DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/wichteln"]
