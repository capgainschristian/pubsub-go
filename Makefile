.PHONY: publisher subscriber logwriter up down
 
PUBLISHER_IMAGE := pubsub-go/publisher
SUBSCRIBER_IMAGE := pubsub-go/subscriber
LOGWRITER_IMAGE := pubsub-go/logwriter
 
## Build the publisher Docker image only
publisher:
	docker build -t $(PUBLISHER_IMAGE) -f publisher/Dockerfile .
 
## Build the subscriber Docker image only
subscriber:
	docker build -t $(SUBSCRIBER_IMAGE) -f subscriber/Dockerfile .

logwriter:
	docker build -t $(LOGWRITER_IMAGE) -f subscriber/Dockerfile .

## Build both images, bootstrap .env, and bring the stack up
up: publisher subscriber logwriter
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example — update MATTERMOST_WEBHOOK_URL before proceeding."; \
	fi
	docker compose up --build
 
## Tear down all containers, networks, and volumes
down:
	docker compose down -v
	rm -f .env