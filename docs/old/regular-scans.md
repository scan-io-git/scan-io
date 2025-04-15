To implement regular scans you can reuse k8s' mechanism called CronJob.
This repo provide helm chart `/helm/scanio-main-cronjob`.

Go to `values.yaml` file and update `command` to make what you need.
By default it regularly scans juice-shop projects. Also update `schedule` value as you need.

To install help chart use the following command:  
`helm install scanio-main-cronjob helm/scanio-main-cronjob/`  

To uninstall:  
`helm uninstall scanio-main-cronjob`