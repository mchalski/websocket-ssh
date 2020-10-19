FROM ubuntu:20.04

RUN apt-get update && apt-get install -y vim net-tools 

COPY deviceconnect /usr/bin/

COPY assets /root/assets

CMD ["/usr/bin/deviceconnect"]
