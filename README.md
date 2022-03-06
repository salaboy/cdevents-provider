# Initial setup and design
Currently we have the following things defined in the setup:
1. A local kind cluster that will be used to provision our bootstraping cluster on GKE,
  this can be useful for cases when we want to teardown / reset upstream infra.
    1. Create a local kind cluster with `make orch-cluster` 
    2. Install `crossplane` and `crossplane gcp-provider` on this local kind cluster - `make install-crossplane`
    3. We are now ready to provision our bootstrap GKE cluster and load it with `provider-helm` and `crossplane` itself.
      ```
        make create-bootstrap-cluster
      ```
2. At the end of above steps we should have a GKE Cluster provisioned, and the crossplane chart deployed on this cluster through helm.
3. Get the bootstrap cluster creds locally with `make get-credentials`.
4. Install knative-eventing onto this cluster – `make install-knative-eventing`
5. Install tekton and setup required RBAC – `make install-tekton`

### Use existing cluster as a bootstrap cluster
If we already have a cluster to use as a bootstrap cluster - to host crossplane and other bootstrapping 
utilities, then we can skip a few steps and directly jump to the following:

1. Install crossplane on the bootstrapping cluster - `make install-crossplane`
2. Install tekton - `make install-tekton`
3. Install knative eventing - `make install-knative-eventing`

### Controllers and types
Currently we have all the existing types and controllers defined in `provider-template`
* `ProviderConfig` – will hold configuration for CDEvents like endpoint to connect to, callback URL etc.
* `MyType` - just for reference, will be removed
* `Cluster controller` - controller for the `github.com/crossplane/provider-gcp/apis/container/v1beta2` resource, we would track this and wait for `Status.Conditions[].Ready == True` and trigger the CDEvent for the start

# provider-template

`provider-template` is a minimal [Crossplane](https://crossplane.io/) Provider
that is meant to be used as a template for implementing new Providers. It comes
with the following features that are meant to be refactored:

- A `ProviderConfig` type that only points to a credentials `Secret`.
- A `MyType` resource type that serves as an example managed resource.
- A managed resource controller that reconciles `MyType` objects and simply
  prints their configuration in its `Observe` method.

## Developing

1. Use this repository as a template to create a new one.
1. Find-and-replace `provider-template` with your provider's name.
1. Run `make` to initialize the "build" Make submodule we use for CI/CD.
1. Run `make reviewable` to run code generation, linters, and tests.
1. Replace `MyType` with your own managed resource implementation(s).

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/docs/contributing/provider_development_guide.md
