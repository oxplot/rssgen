FROM golang:1.16-buster

RUN mkdir -p /app
WORKDIR /app
COPY . /app/
RUN go build -o rssgen

FROM debian
RUN apt update && apt install -y ca-certificates
COPY --from=0 /app/rssgen /
CMD /rssgen -config /rssgen.yaml
