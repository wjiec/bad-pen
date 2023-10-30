
init:
	kubebuilder init --owner "Jayson Wang" \
		--domain example.org \
		--repo github.com/wjiec/programming_k8s/discovery

api:
	kubebuilder create api --group discovery --version v1 --kind Rule
