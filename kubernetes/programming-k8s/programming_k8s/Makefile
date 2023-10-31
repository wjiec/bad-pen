
circle-init:
	cd circle && kubebuilder init --owner "Jayson Wang" \
		--domain example.org \
		--repo github.com/wjiec/programming_k8s/circle

circle-api:
	cd circle && kubebuilder create api \
		--group batch \
		--version v1 \
		--kind CronJob
	cd circle && make manifests
