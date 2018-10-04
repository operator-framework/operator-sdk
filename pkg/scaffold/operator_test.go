package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestOperator(t *testing.T) {
	codegen := NewOperatorCodegen(&OperatorInput{ProjectName: appProjectName})
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if operatorExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(operatorExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const operatorExp = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: app-operator
  template:
    metadata:
      labels:
        name: app-operator
    spec:
      containers:
        - name: app-operator
          # Replace this with the built image name
          image: REPLACE_IMAGE
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - app-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "app-operator"
`
