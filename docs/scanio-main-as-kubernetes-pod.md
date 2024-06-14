The main difference between running main scanio process locally vs in kuberntese is that you want to attach custom serviceaccount with rights to create and delete jobs in the same cluster.

Most of it is included in helm chart called `scanio-main`. Inspect these files. Not much code.

What you have to do is run `helm install scanio-main helm/scanio-main/`.  
Get shell in pod `kubectl exec -it test-pod -- bash`.  
When you are inside the pod, you can run scanio with helm runtime. Continue in `docs/helm-runtime.md`, section "Run remote scan with helm runtine".
