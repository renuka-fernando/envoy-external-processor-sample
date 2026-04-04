IMAGE_NAME ?= envoy-static
IMAGE_TAG  ?= latest

.PHONY: build tag push clean

build:
	docker build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) .

tag:
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

push: tag
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

clean:
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
