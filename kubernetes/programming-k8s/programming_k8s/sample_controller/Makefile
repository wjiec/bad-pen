

codegen:
	client-gen -v 7 \
		--go-header-file=$(PWD)/hack/boilerplate.go.txt \
		--input="sample/v1alpha1" \
		--clientset-name="sample" \
		--clientset-path="./pkg/client/clientset_generated" \
		--output-base="./" \
		--output-package="github.com/wjiec/programming_k8s/sample_controller"
