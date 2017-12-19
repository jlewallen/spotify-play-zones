FROM ubuntu:latest
RUN mkdir /app
WORKDIR /app
COPY play-zones /app
COPY tokens.json /app
COPY static /app/static
EXPOSE 9090
ENTRYPOINT [ "/app/play-zones", "--web-server" ]
