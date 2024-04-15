# scavenger-jobs-operator
Scavenger Jobs Operator is a Kubernetes operator that manages the lifecycle of Scavenger Jobs (SJ) in a Kubernetes cluster.

## Description
Cloud environments are notorious for suffering from low utilization versus reservations. Scavenger Job (hereinafter only SJ) is a job that serves to increase the utilization of cluster resources by consuming free resources (which can be reserved, but are not currently used), but if a request for the use of these reserved resources comes to the cluster, SJ is interrupted, frees resources and leaves them to the workload with a higher priority. All SJs have a lower priority than all other types of workloads in the cluster.

## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<YOUR_IMAGE>
```

Image is available at:
- [peterocker/scavenger-jobs:latest](https://hub.docker.com/r/peterocker/scavenger-jobs)
- [cerit.io/xrucek/scavenger-jobs-operator:latest](https://hub.cerit.io/harbor/projects/159/repositories)

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<YOUR_IMAGE>
```

**Deploy the cert-manager with self-signed certificate needed for the webhook:**

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.3/cert-manager.yaml
kubectl create ns scavenger-jobs-operator-system
make deploy IMG=<YOUR_IMAGE>
```

Or ensure you have cert-manager installed.

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**What now?**
The operator is now running in the `scavenger-jobs-operator-system cluster`. You can create a ScavengerJob (SJ) CRD and the operator will create a Job for you. See the [samples](config/samples) for examples.

**Create instances of your solution**

These scripts will help you create/delete multiple jobs/ SJ and mimic the behaviour of a busy cluster.
```sh
./hack/jobrunner.sh 20 //create 20 sleep jobs 1-20
./hack/jobrunner.sh 2 //create anothe 2 sleep jobs 21-22
./hack/jobrunner.sh -s 1 //run 1 SJ
./hack/jobdeleter.sh //delete all jobs with default "infinity" pattern
./hack/jobdeleter.sh sleep //delete jobs with pattern regex: sleep
```

### To Uninstall
**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```


## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

