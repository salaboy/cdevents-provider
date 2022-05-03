# ====================================================================================
# Setup Project
PROJECT_NAME := cdevents-provider
PROJECT_REPO := github.com/salaboy/$(PROJECT_NAME)
CLUSTER_NAME ?= orch-cluster

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# Setup Output
-include build/makelib/output.mk

# Setup Go
NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/cdevents-provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
-include build/makelib/golang.mk

# Setup Kubernetes tools
-include build/makelib/k8s_tools.mk

# Setup Images
DOCKER_REGISTRY ?= crossplane # ishankhare07/cdevents-provider
IMAGES = $(PROJECT_NAME) $(PROJECT_NAME)-controller
-include build/makelib/image.mk

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

crds.clean:
	@$(INFO) cleaning generated CRDs
	@find package/crds -name *.yaml -exec sed -i.sed -e '1,2d' {} \; || $(FAIL)
	@find package/crds -name *.yaml.sed -delete || $(FAIL)
	@$(OK) cleaned generated CRDs

generate: crds.clean

# integration tests
e2e.run: test-integration

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(HELM3)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@$(ROOT_DIR)/cluster/local/integration_tests.sh || $(FAIL)
	@$(OK) integration tests passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/$(PROJECT_NAME) --debug

dev: $(KIND) $(KUBECTL)
	# @$(INFO) Creating kind cluster
	# @$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	# @$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Crossplane CRDs
	@$(KUBECTL) apply -k https://github.com/crossplane/crossplane//cluster?ref=master
	@$(INFO) Installing Provider CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider controllers
	@$(GO) run cmd/cdevents-provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration run crds.clean dev dev-clean

# ====================================================================================
# Special Targets

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special

orch-cluster:
	# kind create cluster --name ${CLUSTER_NAME} --config kind-config.yaml
	vcluster create orchestrator -n orch --expose
	vcluster connect orchestrator --namespace orch

install-crossplane:
	kubectl create namespace crossplane-system
	helm repo add crossplane-stable https://charts.crossplane.io/stable
	helm repo update
	helm install crossplane --namespace crossplane-system crossplane-stable/crossplane
	sleep 40
	kubectl apply -f config/crossplane/gcp/provider.yaml
	sleep 15
	kubectl apply -f config/crossplane/gcp/provider-config.yaml
	kubectl create secret generic gcp-creds -n crossplane-system --from-file=creds=${CRED_PATH}

create-bootstrap-cluster:
	kubectl apply -f config/crossplane/resources/
	echo "setting up helm provider"
	kubectl apply -f config/crossplane/helm/provider.yaml
	sleep 15
	kubectl apply -f config/crossplane/helm/provider-config.yaml
	kubectl apply -f config/crossplane/helm/release.yaml

get-credentials:
	gcloud container clusters get-credentials bootstrap-cluster --zone asia-south1-a --project tonal-baton-181908

install-knative-serving:
	kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.3.2/serving-crds.yaml
	kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.3.2/serving-core.yaml
	# install networking layer | kourier
	kubectl apply -f https://github.com/knative/net-kourier/releases/download/knative-v1.3.0/kourier.yaml
	kubectl patch configmap/config-network \
			--namespace knative-serving \
			--type merge \
			--patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'

install-knative-eventing:
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.3.2/eventing-crds.yaml
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.3.2/eventing-core.yaml
	# install default Channel (messaging) layer | in-mem standalone
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.3.2/in-memory-channel.yaml
	# Install a Broker layer | MT-Channel based
	kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.3.2/mt-channel-broker.yaml

install-tekton:
	echo "installing tekton pipelines"
	kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
	echo "installing tekton triggers"
	kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
	kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml
	echo "applying tekton user, role and rolebinding"
	kubectl apply -f tekton/rbac/admin-role.yaml
	kubectl apply -f tekton/rbac/crb.yaml 
	kubectl apply -f tekton/rbac/trigger-webhook-role.yaml

install-tekton-resources:
	kubectl apply -f tekton/resources/

setup-eventing-broker-and-triggers:
	kubectl apply -f eventing/

install-tools: get-credentials install-knative install-tekton
