FROM golang:alpine
ENV PORT=3001

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o main .

EXPOSE $PORT

CMD [ "./main" ]
