# curl --insecure -v "http://broker-ingress.knative-eventing.svc.cluster.local/default/default" \
# curl --insecure -v http://sockeye.default.34.100.170.210.sslip.io \
curl --insecure -v "https://broker.ishankhare.dev/default/default" \
-X POST \
-H "Ce-Id: create-gke-cluster" \
-H "Ce-Specversion: 1.0" \
-H "Ce-Type: cd.service.deployed.v1" \
-H "Ce-Source: default/hello" \
-H "Host: broker.ishankhare.dev" \
-H "Content-Type: application/json" \
-d '{}'
