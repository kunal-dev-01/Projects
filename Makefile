# ==============================================================================
# Development Environment Variables
# ==============================================================================
NETWORK_NAME = dev_shared_network

POSTGRES_COMPOSE = db/postgresql/docker/dev/docker-compose.dev.yml
MONGO_COMPOSE = db/mongo/docker/dev/docker-compose.dev.yml
REDIS_COMPOSE = redis/docker/dev/docker-compose.dev.yml
RMQ_COMPOSE = rmq/docker/dev/docker-compose.dev.yml

# ==============================================================================
# Network Setup
# ==============================================================================
.PHONY: dev-network
dev-network:
	@docker network inspect $(NETWORK_NAME) >/dev/null 2>&1 || \
		docker network create $(NETWORK_NAME)
	@echo "Network $(NETWORK_NAME) is ready."

# ==============================================================================
# Postgres (Dev)
# ==============================================================================
.PHONY: dev-postgres-up dev-postgres-down dev-postgres-logs
dev-postgres-up: dev-network
	docker compose -f $(POSTGRES_COMPOSE) up -d

dev-postgres-down:
	docker compose -f $(POSTGRES_COMPOSE) down

dev-postgres-logs:
	docker compose -f $(POSTGRES_COMPOSE) logs -f db

# ==============================================================================
# Postgres Migrations (Dev)
# ==============================================================================
.PHONY: dev-migrate dev-migrate-logs
dev-migrate:
	docker compose -f $(POSTGRES_COMPOSE) up migrate

dev-migrate-logs:
	docker compose -f $(POSTGRES_COMPOSE) logs -f migrate

# ==============================================================================
# Mongo (Dev)
# ==============================================================================
.PHONY: dev-mongo-up dev-mongo-down dev-mongo-logs
dev-mongo-up: dev-network
	docker compose -f $(MONGO_COMPOSE) up -d

dev-mongo-down:
	docker compose -f $(MONGO_COMPOSE) down

dev-mongo-logs:
	docker compose -f $(MONGO_COMPOSE) logs -f

# ==============================================================================
# Redis (Dev)
# ==============================================================================
.PHONY: dev-redis-up dev-redis-down dev-redis-logs
dev-redis-up: dev-network
	docker compose -f $(REDIS_COMPOSE) up -d

dev-redis-down:
	docker compose -f $(REDIS_COMPOSE) down

dev-redis-logs:
	docker compose -f $(REDIS_COMPOSE) logs -f

# ==============================================================================
# RabbitMQ (Dev)
# ==============================================================================
.PHONY: dev-rmq-up dev-rmq-down dev-rmq-logs
dev-rmq-up: dev-network
	docker compose -f $(RMQ_COMPOSE) up -d

dev-rmq-down:
	docker compose -f $(RMQ_COMPOSE) down

dev-rmq-logs:
	docker compose -f $(RMQ_COMPOSE) logs -f

# ==============================================================================
# All Services (Dev)
# ==============================================================================
.PHONY: dev-up dev-down dev-logs
dev-up: dev-network dev-postgres-up dev-mongo-up dev-redis-up dev-rmq-up
	@echo "All DEV services started successfully."

dev-down: dev-postgres-down dev-mongo-down dev-redis-down dev-rmq-down
	@echo "All DEV services stopped."

dev-logs:
	docker compose -f $(POSTGRES_COMPOSE) -f $(MONGO_COMPOSE) -f $(REDIS_COMPOSE) -f $(RMQ_COMPOSE) logs -f
