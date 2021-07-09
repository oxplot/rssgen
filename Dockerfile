FROM golang:1.16-buster

RUN mkdir -p /app
WORKDIR /app
COPY . /app/
RUN go build -o rssgen

FROM ubuntu
COPY --from=0 /app/rssgen /
CMD /rssgen -config /rssgen.yaml
