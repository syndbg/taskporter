FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /root/

COPY taskporter /usr/local/bin/taskporter

RUN chmod +x /usr/local/bin/taskporter

ENTRYPOINT ["taskporter"]
