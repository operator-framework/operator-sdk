FROM osdk-builder as builder

RUN ci/tests/scaffolding/e2e-helm-scaffold.sh

FROM osdk-helm

COPY --from=builder /helm/nginx-operator/watches.yaml ${HOME}/watches.yaml
COPY --from=builder /helm/nginx-operator/helm-charts/ ${HOME}/helm-charts
