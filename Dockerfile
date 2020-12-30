FROM alpine:3.12.3

LABEL maintainer="Felipe Cruz"

ENV TERRAFORM_VERSION=0.14.3

RUN apk update
RUN apk add curl unzip

RUN curl -sSL https://cli.openfaas.com | sh

RUN wget https://releases.hashicorp.com/terraform/0.14.3/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    mv terraform /usr/local/bin && \
    rm -rf terraform_${TERRAFORM_VERSION}_linux_amd64.zip

WORKDIR /work

COPY . .
