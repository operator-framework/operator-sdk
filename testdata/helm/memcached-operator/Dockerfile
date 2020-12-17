# Build the manager binary
FROM quay.io/operator-framework/helm-operator:v1.3.0

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts  ${HOME}/helm-charts
WORKDIR ${HOME}
