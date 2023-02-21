package providers

import (
	"golanglearning/new_project/virtual-kubelet-practice/pkg/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	"time"
	"context"
)

func CreateContainer(pod *v1.Pod, podSandboxId string) error {
	podId := podSandboxId

	config := &v1alpha2.ContainerConfig{}
	klog.Infof("aaa", config)
	err := YamlFile2Struct("./test/example_container.yaml", config)
	if err != nil {
		klog.Error(err)
		return err
	}

	config.Metadata.Name = pod.Spec.Containers[0].Name
	config.Image.Image = pod.Spec.Containers[0].Image
	config.Command = pod.Spec.Containers[0].Command


	ctx, cancel := context.WithTimeout(context.Background(),time.Second*10)
	defer cancel()

	// POD sandbox对应的配置对象
	pConfig := &v1alpha2.PodSandboxConfig{}
	klog.Infof("aaa", pConfig)
	err = YamlFile2Struct("./test/example_sandbox.yaml", pConfig)
	if err != nil  {
		klog.Error(err)
		return err
	}
	pConfig.Metadata.Namespace = pod.Namespace
	pConfig.Metadata.Name = pod.Name
	pConfig.LogDirectory =  "/root/temp"
	pConfig.PortMappings[0].ContainerPort = pod.Spec.Containers[0].Ports[0].ContainerPort


	req1 := &v1alpha2.CreateContainerRequest{
		PodSandboxId: podId,//必须要传
		Config: config, //容器配置
		SandboxConfig: pConfig,//pod配置 。必须要传
	}

	runtimeService := common.NewRuntimeService()
	rsp1, err := runtimeService.CreateContainer(ctx, req1)
	if err != nil {
		klog.Error(err)
		return err
	}

	// 启动容器
	sreq := &v1alpha2.StartContainerRequest{ContainerId: rsp1.ContainerId}

	_, err = runtimeService.StartContainer(ctx, sreq)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}
