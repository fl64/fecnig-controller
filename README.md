```bash
kubectl apply -f k8s/tests/ngc.yaml # enable watchdog with deckhouse NGC
kubectl apply -f k8s/tests/cnp.yaml

# set label `killme: "true"` to any fencing-agent pod 
```

## TODO

- [ ] If someone deletes the `node-manager.deckhouse.io/fencing-enabled` tag, does it need to be restored?