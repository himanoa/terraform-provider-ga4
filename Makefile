.PHONY: build install test testacc fmt vet

build:
	go build -o terraform-provider-ga4 .

# $GOBIN（既定 ~/go/bin）へ入れる。~/.terraformrc の dev_overrides からここを指す
install:
	go install .

test:
	go test ./...

# 実 API を叩くテスト。TF_ACC=1 と GA4 の編集権限を持つ資格情報が要る
testacc:
	TF_ACC=1 go test ./... -v -timeout 30m

fmt:
	gofmt -w .

vet:
	go vet ./...
