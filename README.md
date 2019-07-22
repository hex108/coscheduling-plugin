# Coscheduling plugin

Based on the great work of Kubernetes [new scheduler framework](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/20180409-scheduling-framework.md) and [kube-batch](https://github.com/kubernetes-sigs/kube-batch), it is very promising to impelment [coscheduling](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling) feature(or named [Gang scheduling](https://en.wikipedia.org/wiki/Gang_scheduling)). And this coscheduling plugin is for this purpose. It is just a **POC**(proof of concept) now. We will design and discuss more about it.

## Design and implementation

It is implemented using [PodGroup](https://github.com/kubernetes-sigs/kube-batch/blob/master/pkg/apis/scheduling/v1alpha2/types.go#L89) in kube-batch and [permit plugin](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L202) in Kubernetes new framework now.  In the futher, we might re-design a new `PodGroup` that includes `WaitTimeout` and other attributes.

# How to run

1. Run a kubernetes cluster
  
   We could run a Kubernetes cluster easily in a local machine. Or we could run this coschedling pluing in an existing Kuberntes cluster.
   
    ```
   # hack/local-up-cluster.sh        # run it in the Kuberentes repo directory
    ```

2. Install CRD PodGroup

    ```
    # kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/kube-batch/master/deployment/kube-batch/templates/scheduling_v1alpha2_podgroup.yaml
    ```

3. Kill the default scheduler, and run the scheduler with coscheduling plugin

    ```
    # kill XXX   # XXX is the pid of the default scheduler
    # go build -o ./coscheduling ./main.go
    # ./coscheduling --v=5 --config ./hack/config.json --leader-elect=false --kubeconfig /var/run/kubernetes/scheduler.kubeconfig --feature-gates=AllAlpha=false --master=https://localhost:6443
    ```

4. Test

    ```
    # kubectl create -f examples/podgroup.yaml
    # kubectl create -f ./examples/deployment.yaml
    ```
    
    In the scheduler log, we'll see logs as following:
    
    ```
    Wait for pod number of PodGroup to be 3, got 1 now
    ...
    Wait for pod number of PodGroup to be 3, got 2 now
    ...
    Wait for pod number of PodGroup to be 3, got 3 now
    ```
    
    When it reaches the min number 3, it will permit these waiting pods.
    
    We could also test other cases for it, will add more tests for it.
    
    
