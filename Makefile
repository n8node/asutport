.PHONY: dev prod down test lint logs logs-prod psql wp-cli deploy backup-db backup-wp status golden setup

dev:
	docker compose --profile dev up --build -d

prod:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d

down:
	docker compose --profile dev down

logs:
	docker compose --profile dev logs -f

logs-prod:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

status:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml ps

test:
	cd backend && go test ./...

lint:
	cd backend && go vet ./...

psql:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

wp-cli:
	docker compose exec wordpress wp --allow-root

deploy:
	./scripts/deploy.sh main

backup-db:
	docker compose exec -T postgres pg_dump -U $(POSTGRES_USER) $(POSTGRES_DB) \
		| gzip > backups/pg_$(shell date +%Y%m%d_%H%M%S).sql.gz

backup-wp:
	docker compose exec -T mysql mysqldump -u $(WP_DB_USER) -p$(WP_DB_PASSWORD) $(WP_DB_NAME) \
		| gzip > backups/wp_$(shell date +%Y%m%d_%H%M%S).sql.gz

golden:
	@echo "TODO: golden set runner (Phase 4+)"

setup:
	./scripts/setup.sh
