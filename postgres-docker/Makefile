.PHONY: docker-build run-dev run-prod

IMAGE_NAME=ghs-dbi
CONTAINER_NAME=ghs-db
CONTAINER_NAME_DEV=ghs-db-dev

docker-build:
	docker build -t $(IMAGE_NAME) .

run-dev:
	docker run -d -p 5432:5432 --name $(CONTAINER_NAME_DEV) $(IMAGE_NAME)

run-prod:
	docker run -d --name $(CONTAINER_NAME) $(IMAGE_NAME)
