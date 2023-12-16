default: tests

ifndef UNLEASH_DOCKER_IMAGE
override UNLEASH_DOCKER_IMAGE = ghcr.io/ivarconr/unleash-enterprise:latest
endif

.PHONY: tests
tests: clean start-docker
	@echo "Running tests..."
	@bash tests.sh
	@echo "You can check the results at http://localhost:4242"
	@echo "To clean up run 'make clean'"

.PHONY: start-docker
start-docker:
	@echo "Starting docker..."
	@UNLEASH_DOCKER_IMAGE=$(UNLEASH_DOCKER_IMAGE) docker compose up -d
	@echo "Waiting for Unleash to be healthy..."
	@while ! curl -s "http://localhost:4242/health"; do sleep 0.1; done
	@echo "\nReady!"


.PHONY: clean-docker
clean-docker: 
	@echo "Cleaning up docker..."
	@docker compose stop && docker compose rm -f

.PHONY: clean-state
clean-state: 
	@echo "Cleaning up Terraform state..."
	@rm -f terraform.tfstate*

.PHONY: clean-provider
clean-provider: 
	@echo "Cleaning up Terraform providers..."
	@rm -rf stage_*/.terraform*

.PHONY: set-provider
set-provider: clean-provider
	@echo "Setting Terraform provider to $(VERSION)"
	@sed -i 's/^\(\s*\)version = ".*"/\1version = "$(VERSION)"/g' stage_*/provider.tf

.PHONY: clean
clean: clean-docker clean-state