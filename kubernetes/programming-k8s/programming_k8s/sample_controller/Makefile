.PHONY: codegen

test: .gen-register

# go install k8s.io/code-generator/cmd/register-gen@latest
.gen-register:
	register-gen \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--input-dirs=./pkg/apis/sample/v1alpha1 \
		--output-base=. \
		--output-file-base=zz_generated.register

# go install k8s.io/code-generator/cmd/deepcopy-gen@latest
.gen-deepcopy:
	deepcopy-gen \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--input-dirs=./pkg/apis/sample/v1alpha1 \
		--output-base=. \
		--output-file-base=zz_generated.deepcopy

codegen: .gen-register .gen-deepcopy
	client-gen -v 7 \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--clientset-name="sample" \
		--input-dirs=./pkg/apis/sample/v1alpha1 \
		--output-base=. \
		--output-package=pkg/client
