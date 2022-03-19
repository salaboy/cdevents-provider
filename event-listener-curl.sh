curl -v "http://el-gke-provision-listener.default.svc.cluster.local:8080" \
-X POST \
-H "Ce-Id: say-hello-goodbye" \
-H "Ce-Specversion: 1.0" \
-H "Ce-Type: greeting" \
-H "Ce-Source: sendoff" \
-H "Content-Type: application/json" \
-d '{"path": "workspace/source/config/crossplane/resources/", "git_source_name": "cluster-git"}'
