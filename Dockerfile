FROM alpine:3.17
COPY eks-auto-updater /opt/eks-auto-updater/bin/eks-auto-updater
WORKDIR /opt/eks-auto-updater
