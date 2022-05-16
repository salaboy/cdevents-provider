# curl --insecure -v "http://broker-ingress.knative-eventing.svc.cluster.local/default/default" \
curl --insecure -v "https://broker.ishankhare.dev/default/default" \
-X POST \
-H "Ce-Id: create-gke-cluster" \
-H "Ce-Specversion: 1.0" \
-H "Ce-Type: cd.service.deployed.v1" \
-H "Ce-Source: default/hello" \
-H "Content-Type: application/json" \
-H "Host: broker.ishankhare.dev" \
-d '{}'
