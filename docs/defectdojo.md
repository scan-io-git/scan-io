
# Installation

If you have DefectDojo installed you can skip this chapter.  

If you want to install DefectDojo in your k8s cluster, the easiest way is to use its helm chart. The following instructions will show how to create dev instance of defectdojo, without SSO, TLS certificate, ingress and persistent storage. But this instruction will be enough to run defectdojo for dev or testing purposes.
1. Install defectdojo's repository to your helm CLI.
```bash
# install repository
❯ helm repo add defectdojo 'https://raw.githubusercontent.com/DefectDojo/django-DefectDojo/helm-charts'

❯ helm repo list
NAME            URL                                                                       
defectdojo      https://raw.githubusercontent.com/DefectDojo/django-DefectDojo/helm-charts

# update repositories charts
❯ helm repo update

# check that you helm CLI sees the chart
❯ helm search repo defectdojo
NAME                    CHART VERSION   APP VERSION     DESCRIPTION                                      
defectdojo/defectdojo   1.6.43          2.15.1          A Helm chart for Kubernetes to install DefectDojo
```
2. Install helm chart
```bash
helm install defectdojo defectdojo/defectdojo \
    --set createSecret=true \
    --set createRedisSecret=true \
    --set createPostgresqlSecret=true \
    --set host=defectdojo.example.com \
    --set django.ingress.enabled=false \
    --set django.ingress.activateTLS=false \
    --set celery.broker=redis \
    --set postgresql.primary.persistence.enabled=false \
    --set rabbitmq.enabled=false \
    --set redis.enabled=true \
    --set redis.persistence.enabled=false
```
3. Run port-forwarding.
```bash
kubectl port-forward --namespace=default service/defectdojo-django 8080:80
```
4. Update `/etc/hosts` file:
```
::1       defectdojo.example.com
127.0.0.1 defectdojo.example.com
```
5. Get Admin Password
```
echo "DefectDojo admin password: $(kubectl \
      get secret defectdojo \
      --namespace=default \
      --output jsonpath='{.data.DD_ADMIN_PASSWORD}' \
      | base64 --decode)"
```
6. Visit `http://defectdojo.example.com:8080`


# Push results to defectdojo