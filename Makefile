TAG=v0.0.0-dev0

.PHONY: build
build:
	werf build --repo docker.io/fl64/fencing-agent --add-custom-tag=$(TAG)