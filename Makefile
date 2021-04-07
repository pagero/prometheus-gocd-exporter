IMAGE?=pagero/prometheus-gocd-exporter
TAG?=latest

.PHONY: docker
docker:
	docker build -t $(IMAGE):$(TAG) .

.PHONY: test
test:
	go test -v

.PHONY: release
release: docker
	docker push $(IMAGE):$(TAG)
