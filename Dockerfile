# env GOOS=linux GOARCH=386 go build -o todo-linux
# docker build . -t <image_name>
# docker run -d --name todo -v /tmp/db/:/usr/share/app/db/ -p 7540:7540 <image_name>

FROM ubuntu:latest

ENV TODO_DBFILE=/usr/share/app/db/todo.db
ENV TODO_PASSWORD=test
EXPOSE 7540

RUN mkdir -p /usr/share/app/db
WORKDIR /usr/share/app
ADD ./web ./web
ADD ./todo-linux .
CMD ./todo-linux