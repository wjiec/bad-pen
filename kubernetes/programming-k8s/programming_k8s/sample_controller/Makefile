.DEFAULT_GOAL: codegen

test: .gen_register

codegen: .gen_register
	client-gen -v 7 \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--clientset-name=samplerclientset \
		--input=sample/v1alpha1 \
		--input-base=github.com/wjiec/programming_k8s/sample_controller/pkg/apis \
		--output-base=. \
		--output-package=github.com/wjiec/programming_k8s/sample_controller/pkg/client/clientset_generated/ \
		--trim-path-prefix=github.com/wjiec/programming_k8s/sample_controller

.PHONY: .gen_register
.gen_register:
	register-gen \
		--output-file-base=zz_generated.register \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--output-base=$(PWD) \
		--input-dirs=github.com/wjiec/programming_k8s/sample_controller/pkg/apis/v1alpha1
