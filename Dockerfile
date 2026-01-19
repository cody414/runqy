FROM golang:1.25.3

# copy app
COPY ./app /app
WORKDIR /app

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /runqy

EXPOSE 3000

ENTRYPOINT [ "/runqy" ]
