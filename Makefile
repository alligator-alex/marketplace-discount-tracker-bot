BOT_SH=docker compose exec bot sh -c
BOT_EXECUTABLE=/slodych/bot

include .env

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1;32m%-25s\033[0m %s\n", $$1, $$2}'

prepare-and-up: ## First lauch
	mkdir -p .database
	make up
	make migrate

up: ## Run Docker containers
	docker compose up --build -d --remove-orphans

down: ## Stop Docker containers
	docker compose down

cli: ## Run console command (make cli command="<command>")
ifndef command
	$(error command not specified (make cli command command=""))
endif
ifdef path
	$(BOT_SH) '$(BOT_EXECUTABLE) impulse101 $(command)'
endif
