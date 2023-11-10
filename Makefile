TAG=v0.0.1-dev0

.PHONY: build deploy undeploy enable-watchdog
build:
	werf build --repo docker.io/fl64/fencing-agent --add-custom-tag=$(TAG)

deploy:
	kubectl apply -k k8s/

undeploy:
	kubectl delete -k k8s/

ud: undeploy deploy

enable-watchdog:
	kubectl apply -f k8s/tests/ngc.yaml

enable-api:
	kubectl delete -f k8s/tests/cnp.yaml

disable-api:
	kubectl apply  -f k8s/tests/cnp.yaml

