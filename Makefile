export K8S_VERSION ?= 1.23.x
export KUBEBUILDER_ASSETS ?= ${HOME}/.kubebuilder/bin

## Inject the app version into project.Version
LDFLAGS ?= -ldflags=-X=github.com/bwagner5/karpenter-k3d/pkg/utils/project.Version=$(shell git describe --tags --always)
GOFLAGS ?= $(LDFLAGS)
WITH_GOFLAGS = GOFLAGS="$(GOFLAGS)"

## Extra helm options
CLUSTER_NAME ?= my-karpenter-cp
CLUSTER_ENDPOINT ?= "http://localhost"
HELM_OPTS ?= --set clusterName=${CLUSTER_NAME} \
			--set clusterEndpoint=${CLUSTER_ENDPOINT} \
			--set env[0].K3D_HELPER_IMAGE_TAG='5.4.4'
TEST_FILTER ?= .*

# CR for local builds of Karpenter
KO_DOCKER_REPO ?= ko.local

help: ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

dev: verify test ## Run all steps in the developer loop

ci: toolchain verify licenses  ## Run all steps used by continuous integration

test: ## Run tests
	go test -run=${TEST_FILTER} ./pkg/...

verify: codegen ## Verify code. Includes dependencies, linting, formatting, etc
	go mod tidy
	go mod download
	golangci-lint run
	@git diff --quiet ||\
		{ echo "New file modification detected in the Git working tree. Please check in before commit.";\
		if [ $(MAKECMDGOALS) = 'ci' ]; then\
			exit 1;\
		fi;}

licenses: ## Verifies dependency licenses
	go mod download
	! go-licenses csv ./... | grep -v -e 'MIT' -e 'Apache-2.0' -e 'BSD-3-Clause' -e 'BSD-2-Clause' -e 'ISC' -e 'MPL-2.0'


build: ## Build the Karpenter controller and webhook images using ko build
	$(eval CONTROLLER_IMG=$(shell KO_DOCKER_REPO=$(KO_DOCKER_REPO) $(WITH_GOFLAGS) ko build -B github.com/bwagner5/karpenter-k3d/cmd/controller))
	$(eval WEBHOOK_IMG=$(shell KO_DOCKER_REPO=$(KO_DOCKER_REPO) $(WITH_GOFLAGS) ko build -B github.com/bwagner5/karpenter-k3d/cmd/webhook))
	k3d image import $(CONTROLLER_IMG)
	k3d image import $(WEBHOOK_IMG)

apply: build ## Deploy the controller from the current state of your git repository into your ~/.kube/config cluster
	helm upgrade --create-namespace --install karpenter-k3d ~/git/karpenter/charts/karpenter --namespace karpenter \
		$(HELM_OPTS) \
		--set controller.image=$(CONTROLLER_IMG) \
		--set webhook.image=$(WEBHOOK_IMG)

install:  ## Deploy the latest released version into your ~/.kube/config cluster
	@echo Upgrading to $(shell grep version charts/karpenter/Chart.yaml)
	helm upgrade --install karpenter-k3d charts/karpenter --namespace karpenter \
		$(HELM_OPTS)

delete: ## Delete the controller from your ~/.kube/config cluster
	helm uninstall karpenter-k3d --namespace karpenter

codegen: ## Generate code. Must be run if changes are made to ./pkg/apis/...
	controller-gen \
		object:headerFile="hack/boilerplate.go.txt" \
		crd \
		paths="./pkg/..." \
		output:crd:artifacts:config=charts/karpenter/crds
	hack/boilerplate.sh

release: release-gen ## Generate release manifests and publish a versioned container image.
	$(WITH_GOFLAGS) ./hack/release.sh

nightly: ## Tag the latest snapshot release with timestamp
	./hack/add-snapshot-tag.sh $(shell git rev-parse HEAD) $(shell date +"%Y%m%d") "nightly"

snapshot: ## Generate a snapshot release out of the current commit
	$(WITH_GOFLAGS) ./hack/snapshot.sh

stablerelease: ## Tags the snapshot release of the current commit with the latest tag available, for prod launch
	./hack/add-snapshot-tag.sh $(shell git rev-parse HEAD) $(shell git describe --tags --exact-match || echo "Current commit is not tagged") "stable"

toolchain: ## Install developer toolchain
	./hack/toolchain.sh

.PHONY: help dev ci release test battletest verify codegen docgen apply delete toolchain release release-gen licenses issues website nightly snapshot e2etests
