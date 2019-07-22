package coscheduling

import (
	"encoding/json"
	"fmt"
	"time"

	kbapiv1alpha2 "github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha2"
	kbver "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned"
	kbinfo "github.com/kubernetes-sigs/kube-batch/pkg/client/informers/externalversions"
	kbclientv1alpha2 "github.com/kubernetes-sigs/kube-batch/pkg/client/listers/scheduling/v1alpha2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

type CoSchedulingPlugin struct {
	FrameworkHandler framework.FrameworkHandle
	PodGroupLister   kbclientv1alpha2.PodGroupLister
}

type Configuration struct {
	KubeMaster string `json:"kube_master,omitempty"`
	KubeConfig string `json:"kube_config,omitempty"`
}

var _ = framework.PermitPlugin(&CoSchedulingPlugin{})

// Name is the name of the plug used in Registry and configurations.
const Name = "coscheduling"

// Name returns name of the plugin. It is used in logs, etc.
func (cs *CoSchedulingPlugin) Name() string {
	return Name
}

func (cs *CoSchedulingPlugin) Permit(pc *framework.PluginContext, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	groupName, exist := pod.Annotations[kbapiv1alpha2.GroupNameAnnotationKey]
	if !exist {
		return framework.NewStatus(framework.Success, ""), 0
	}
	podGroup, err := cs.PodGroupLister.PodGroups(pod.Namespace).Get(groupName)
	if err != nil {
		klog.Errorf("Failed to get PodGroup %s for pod %s/%s: %v", groupName, pod.Namespace, pod.Name)
		return framework.NewStatus(framework.Success, ""), 0
	}

	// MinMember defines the minimal number of pods to run
	if podGroup.Spec.MinMember <= 1 {
		return framework.NewStatus(framework.Success, ""), 0
	}

	count := int32(1)
	search := func(p framework.WaitingPod) {
		// TODO: add more checks for these pods, e.g. whether it has been deleted
		if p.GetPod().Annotations[kbapiv1alpha2.GroupNameAnnotationKey] == groupName {
			count++
		}
	}
	cs.FrameworkHandler.IterateOverWaitingPods(search)

	if count < podGroup.Spec.MinMember {
		klog.V(4).Infof("Wait for pod number of PodGroup to be %d, got %d now", podGroup.Spec.MinMember, count)
		return framework.NewStatus(framework.Wait, ""), 1 * time.Minute
	}

	allow := func(p framework.WaitingPod) {
		if p.GetPod().Annotations[kbapiv1alpha2.GroupNameAnnotationKey] == groupName {
			p.Allow()
		}
	}
	cs.FrameworkHandler.IterateOverWaitingPods(allow)

	return nil, 0
}

// New initializes a new plugin and returns it.
func New(configuration *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {
	var config Configuration
	// TODO: decode it in a better way
	if err := json.Unmarshal(configuration.Raw, &config); err != nil {
		klog.Errorf("Failed to decode %+v: %v", configuration.Raw, err)
		return nil, fmt.Errorf("failed to decode configuration: %v", err)
	}

	klog.V(4).Infof("Plugin %s's config: master(%s), kube-config(%s)", Name, config.KubeMaster, config.KubeConfig)
	// Init kube-batch client and PodGroupInformer
	c, err := clientcmd.BuildConfigFromFlags(config.KubeMaster, config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init rest.Config: %v", err)
	}
	kbClient := kbver.NewForConfigOrDie(c)
	kbinformer := kbinfo.NewSharedInformerFactory(kbClient, 0)
	// create informer for PodGroup information
	podGroupLister := kbinformer.Scheduling().V1alpha2().PodGroups().Lister()

	go kbinformer.Start(nil)

	// TODO: wait for kbinformer cache synced

	return &CoSchedulingPlugin{FrameworkHandler: f, PodGroupLister: podGroupLister}, nil
}
