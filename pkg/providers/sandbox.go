package providers

import (
	"context"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/helper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	"time"
)

// TODO: 删除沙箱
func DeleteContainer() {

}

func CreateSandbox(pod *v1.Pod) (string, error) {
	config1 := &v1alpha2.PodSandboxConfig{}
	err := helper.YamlFile2Struct("./test/example_sandbox.yaml", config1)
	klog.Info(config1)
	if err != nil {
		klog.Error(err)
		return "", nil
	}

	config1.Metadata.Namespace = pod.Namespace
	config1.Metadata.Name = pod.Name
	config1.LogDirectory = "/root/temp"
	a := &v1alpha2.PortMapping{}
	config1.PortMappings = append(config1.PortMappings, a)
	config1.PortMappings[0].ContainerPort = pod.Spec.Containers[0].Ports[0].ContainerPort

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	req := &v1alpha2.RunPodSandboxRequest{
		Config: config1,
	}
	rsp, err := common.NewRuntimeService().RunPodSandbox(ctx, req)
	if err != nil {
		klog.Error(err)
		return "", nil
	}
	klog.Infof(rsp.PodSandboxId)
	return rsp.PodSandboxId, nil
}
