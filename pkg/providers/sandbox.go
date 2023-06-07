package providers

import (
	"context"
	"golanglearning/new_project/virtual-kubelet-practice/pkg/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	"time"
)

func CreateSandbox(pod *v1.Pod) (string, error) {
	config1 := &v1alpha2.PodSandboxConfig{}
	err := YamlFile2Struct("./test/example_sandbox.yaml", config1)
	klog.Info(config1)
	if err != nil {
		klog.Error(err)
		return "", nil
	}

	config1.Metadata.Namespace = pod.Namespace
	config1.Metadata.Name = pod.Name
	config1.LogDirectory = "/root/temp"
	//config1.PortMappings[0].ContainerPort = pod.Spec.Containers[0].Ports[0].ContainerPort

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
