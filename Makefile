include .env

POSTGRESQL_URL='postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${HOST_DB}:${PORT_DB}/${POSTGRES_DB}?sslmode=disable'

.DEFAULT_GOAL := help

.PHONY: help
help: ## display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

PHONY: migr_cr 
migr_cr: ### create new migration titled NAMEM (run NAMEM=some name make migr_cr)
	migrate -verbose create -ext sql -dir ./migrations -seq $$NAMEM

PHONY: migr_up  
migr_up: ### migration up
	migrate -verbose -database ${POSTGRESQL_URL} -path ./migrations up 
	
PHONY: migr_down  
migr_down: ### migration down
	migrate -verbose -database ${POSTGRESQL_URL} -path ./migrations down

PHONY: migr_ver 
migr_ver: ### print current migration version
	migrate -verbose -database ${POSTGRESQL_URL} -path ./migrations version
PHONY: doc
doc: ### create docs.go
	~/go/bin/swag init -g internal/crm/crm.go
