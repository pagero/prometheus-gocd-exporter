IMAGE?=pagero/prometheus-gocd-exporter
TAG?=latest

.PHONY: docker
docker: deps
	docker build -t $(IMAGE):$(TAG) .

.PHONY: release
release: docker
	docker push $(IMAGE):$(TAG)

.PHONY: deps
deps:
	dep ensure -update
