module github.com/salaboy/cdevents-provider

go 1.16

require (
	github.com/crossplane/crossplane-runtime v0.15.1-0.20211202230900-d43d510ec578
	github.com/crossplane/crossplane-tools v0.0.0-20210916125540-071de511ae8e
	github.com/crossplane/provider-gcp v0.20.0
	github.com/google/go-cmp v0.5.6
	github.com/pkg/errors v0.9.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/controller-tools v0.6.2
)
