FROM golang:1.23.3-alpine as build

ENV GO111MODULE=on
ENV CGO_ENABLED=0

WORKDIR /home/app

COPY go.mod /home/app
COPY go.sum /home/app

RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

RUN templ generate
RUN go build -ldflags '-s' -o /home/app/bin/blogs cmd/main.go

FROM alpine:latest AS final

RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        ca-certificates \
        tzdata \
        && \
        update-ca-certificates

COPY --from=build /home/app/bin/blogs /bin/blogs
COPY --from=build /home/app/static /static

EXPOSE 8001
