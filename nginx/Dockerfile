FROM alpine:latest

RUN apk upgrade && apk update && apk add nginx

WORKDIR /var/www/nginx

COPY /www .

COPY nginx.conf /etc/nginx/nginx.conf

CMD ["nginx","-g","pid /tmp/nginx.pid;"]

EXPOSE 80/tcp
