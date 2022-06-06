# Build the manager binary
FROM quay.io/operator-framework/helm-operator:unknown

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts  ${HOME}/helm-charts
WORKDIR ${HOME}
