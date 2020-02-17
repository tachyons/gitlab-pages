FROM golang:1.13.7

WORKDIR app
EXPOSE 3010
COPY . /go/app/

RUN make gitlab-pages
