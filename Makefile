IMAGE?=pagero/prometheus-gocd-exporter
TAG?=latest

.PHONY: docker
docker:
	docker build -t $(IMAGE):$(TAG) .

release: docker
	docker push $(IMAGE):$(TAG)
