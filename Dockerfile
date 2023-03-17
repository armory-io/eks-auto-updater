FROM alpine:3.17
LABEL org.opencontainers.image.source https://github.com/armory-io/eks-auto-updater
COPY eks-auto-updater /opt/eks-auto-updater/bin/eks-auto-updater
WORKDIR /opt/eks-auto-updater