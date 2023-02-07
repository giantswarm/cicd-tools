.PHONY: docker-build-%
docker-build-%: # Build the docker file of a specific directory
	@echo "Building '$*'..."
	@cd "$*" && docker build --platform linux/amd64 -t quay.io/giantswarm/$*:dev .
