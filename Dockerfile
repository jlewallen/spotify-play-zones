FROM ubuntu:latest
RUN mkdir /app
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates
COPY play-zones /app
COPY tokens.json /app
COPY static /app/static
RUN ls -alh /app
EXPOSE 9090
ENTRYPOINT [ "/app/play-zones", "--web-server" ]
