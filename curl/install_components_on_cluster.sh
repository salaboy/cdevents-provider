# curl --insecure -v "http://broker-ingress.knative-eventing.svc.cluster.local/default/default" \
curl --insecure -v "https://broker.ishankhare.dev/default/default" \
-X POST \
-H "Ce-Id: setup-gke-cluster" \
-H "Ce-Specversion: 1.0" \
-H "Ce-Type: cd.environment.created.v1" \
-H "Ce-Source: curl" \
-H "Content-Type: application/json" \
-H "Host: broker.ishankhare.dev" \
-d '{
	"path": "workspace/source/config/",
	"git_source_name": "cluster-git",
	"docker_repo": "ishankhare07"
}'
