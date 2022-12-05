# Step by step guide "How to use a local minikube cluster with a Scanio"
*Tested on a MacOS with an M1 chip.*

1. [Install](https://minikube.sigs.k8s.io/docs/start/) minikube.
2. ```minikube start``` - starting our one node cluster.

3. Build and put a docker image with the application.
On this step you have few different options:
- Build a docker container manualy with ```docker build -t scanio .``` or ```make build```. After a building you have to put the image to a minikube context with a ```minikube image load scanio:latest``` command.
- Build a docker container with ```minikube image build -t scanio .```.
- Use ```eval $(minikube docker-env)``` and build a docker container manualy with ```docker build -t scanio .``` or ```make build```.

4. Now you should install a scanio-main helm chart which will create persistent volume mounted to a local cluster disk and start an infinity pod with privileges to PVCs and jobs.
```helm install scanio-main helm/scanio-minikube/scanio-main/```

5. Now you could use scanio 
- ```scanio run ...``` - through wrapper
- ```helm install test-job helm/scanio-job --set command="{'scanio', '--help'}" --set image.repository=scanio --set image.tag=latest --set suffix=test-job``` - manually

You can check your cluster local files by using a ```minicube ssh``` command. All files developed by main pod and jobs will be in a ```/data/scanio/``` directory. After a minikube cluster redeploy/stop your files won't be erased.
You may mount files from your local disk to a minikube file system ```minikube mount ~/.scanio/projects/:/data/scanio``` as well.

# Possible error
- ```Error: failed to start container "test-pod": Error response from daemon: error while creating mount source path '/data/scanio': mkdir /data: file exists```
You should uninstall all helm charts and restart a minikube cluster.

# Additional articles
- Persistent volumes in [Minikube](https://minikube.sigs.k8s.io/docs/handbook/persistent_volumes/)
- How to [create PV claim in K8S](https://kubernetes.io/docs/tasks/configure-pod-container/configure-persistent-volume-storage/)