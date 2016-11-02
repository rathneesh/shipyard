# Deploy ILM using pods

In order to be able to deploy ILM using pods there should be a cluster up and running, also there needs to be [kubernetes] installed.
The next step is to create a clair_config folder in /etc/ and add the config.yaml file, and change the host to localhost.

## Run the following commands to start the pods and services:

```
kubectl create -f rethink-pod.yml
```
```
kubectl create -f backend-srv.yml
```
```
kubectl create -f controller-pod.yml
```

### Verify that ILM is up and running

Access `localhost:8082` from browser.

[kubernetes]: https://blog.jetstack.io/blog/k8s-getting-started-part2/


