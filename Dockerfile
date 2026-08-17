# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG APP=api
RUN CGO_ENABLED=0 go build -o /out/app ./apps/${APP}/cmd

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/app /app
EXPOSE 8080 8081 2525
ENTRYPOINT ["/app"]
