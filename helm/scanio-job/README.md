```bash
helm template scanio-job --set image.repository=$DOCKER_IMAGE,image.tag=latest | kubectl apply -f -
```
