FROM golang AS build-stage

WORKDIR /app

COPY . .
RUN go mod download && go build main.go

FROM ubuntu

WORKDIR /

COPY --from=build-stage app/main /main

ENTRYPOINT ["/main"]