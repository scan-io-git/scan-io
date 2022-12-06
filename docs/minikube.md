
# Prerequisite for minikube (helm charts configuration)
If you would like work with mimikube you will have to setup some values in helm charts.

## Persistent Storage
To deploy persistent storage, you can reuse prepared yaml files. `kubectl apply -f kubernetes/minikube`.
Provide persistent storage claim name to helm value `persistence.claimName`, and set `persistence.enabled: true`.
After work to destroy resources use `kubectl delete -f kubernetes/minikube`.

# Step by step guide "How to use a local minikube cluster with Scanio"
*Tested on a MacOS with an M1 chip.*

1. [Install](https://minikube.sigs.k8s.io/docs/start/) minikube.
2. ```minikube start``` - starting our one node cluster.
3. Build and put a docker image with the application.

On this step you have few different options:
- Build a docker container manualy with ```docker build -t scanio .``` or ```make docker```. After a building you have to put the image to a minikube context with a ```minikube image load scanio:latest``` command. 

Don't forget to load a new image every time after building. 
- Build a docker container with ```minikube image build -t scanio .```.
- Use ```eval $(minikube docker-env)``` and build a docker container manualy with ```docker build -t scanio .``` or ```make docker```. This approach may work not properly.

4. Now you should install a scanio-main helm chart that will create a persistent volume which is mounted to a local cluster disk and start an infinity pod with privileges to PVCs and Jobs.

```helm install scanio-main helm/scanio-minikube/scanio-main/```

5. Now you could use scanio 
- ```scanio run ...``` - through a wrapper.
- ```helm install test-job helm/scanio-minikube/scanio-job --set command="{'scanio', '--help'}" --set image.repository=scanio --set image.tag=latest --set suffix=test-job``` - manually.

You can check your cluster local files by using a ```minikube ssh``` command. All files developed by main pod and jobs will be in a ```/data/scanio/``` directory. If a minikube cluster redeploy or stop your files won't be erased.

You may mount files from your local file system to a minikube file system ```minikube mount ~/.scanio/projects/:/data/scanio``` as well.

# Possible errors
- ```Error: failed to start container "test-pod": Error response from daemon: error while creating mount source path '/data/scanio': mkdir /data: file exists```

You should uninstall all helm charts and restart a minikube cluster.

# Additional articles
- Persistent volumes in [Minikube](https://minikube.sigs.k8s.io/docs/handbook/persistent_volumes/)
- How to [create PV claim in K8S](https://kubernetes.io/docs/tasks/configure-pod-container/configure-persistent-volume-storage/)