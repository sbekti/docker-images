FROM    amazon/aws-cli:2.26.5
LABEL   maintainer=samudra.bekti@gmail.com

RUN     yum install -y tar xz gzip

WORKDIR /aws
ENTRYPOINT ["/usr/local/bin/aws"]
