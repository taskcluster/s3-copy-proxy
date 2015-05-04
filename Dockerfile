FROM ubuntu:14.04
MAINTAINER James Lal [:lightsofapollo] <jlal@mozilla.com>

RUN apt-get update
RUN apt-get install -y ca-certificates
EXPOSE 80
COPY target/s3-copy-proxy /s3-copy-proxy
ENTRYPOINT ["/s3-copy-proxy", "--port", "80"]
