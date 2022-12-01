For minikube 
```
eval $(minikube docker-env)
make docker
minikube image load scanio:latest
helm install scanio-main helm/scanio-main/
kubectl exec -ti test-pod -- bash
helm install 5319969e-eced-45c8-b8aa-2df3542a4d5d /scanio-helm/scanio-job --set command="{scanio,run,--vcs-plugin,github,--vcs-url,github.com,--scanner-plugin,bandit,--repos,juice-shop/juicy-chat-bot,--storage-type,s3,--s3bucket,my-s3-bucket-q97843yt9}" --set image.repository=scanio --set image.tag=latest --set suffix=5319969e-eced-45c8-b8aa-2df3542a4d5d
helm uninstall scanio-main
```