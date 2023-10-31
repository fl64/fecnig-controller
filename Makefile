TAG=v0.0.0-dev0

.PHONY: build deploy undeploy
build:
	werf build --repo docker.io/fl64/fencing-agent --add-custom-tag=$(TAG)

deploy:
	kubectl apply -k k8s/


undeploy:
	kubectl delete -k k8s/
