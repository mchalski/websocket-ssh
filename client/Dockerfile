FROM ubuntu:20.04

RUN apt-get update && apt-get install -y vim net-tools 

COPY client /usr/bin/

COPY entrypoint /usr/bin/

ENTRYPOINT ["/usr/bin/entrypoint"]
