#!/bin/bash

set -x
set -e

# install CRD PodGroup
kubectl apply -f https://github.com/kubernetes-sigs/kube-batch/blob/master/deployment/kube-batch/templates/scheduling_v1alpha2_podgroup.yaml

# start scheduler
go build -o ./coscheduling ./main.go
./coscheduling --v=5 --leader-elect=false --kubeconfig /var/run/kubernetes/scheduler.kubeconfig --feature-gates=AllAlpha=false --master=https://localhost:6443