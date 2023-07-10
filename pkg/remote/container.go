package remote

import (
	"context"
	"fmt"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	v1 "k8s.io/api/core/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// CreateContainer 创建容器
func CreateContainer(ctx context.Context, client criapi.RuntimeServiceClient, config *criapi.ContainerConfig, podConfig *criapi.PodSandboxConfig, pId string) (string, error) {

	request := &criapi.CreateContainerRequest{
		PodSandboxId:  pId,
		Config:        config,
		SandboxConfig: podConfig,
	}

	r, err := client.CreateContainer(context.Background(), request)

	if err != nil {

		return "", err
	}
	return r.ContainerId, nil
}

// StartContainer 启动容器
func StartContainer(ctx context.Context, client criapi.RuntimeServiceClient, cId string) error {

	if cId == "" {
		err := errdefs.InvalidInput("ID cannot be empty")

		return err
	}
	request := &criapi.StartContainerRequest{
		ContainerId: cId,
	}

	_, err := client.StartContainer(context.Background(), request)

	if err != nil {

		return err
	}

	return nil
}

// GetContainerCRIStatus 获取容器状态
func GetContainerCRIStatus(ctx context.Context, client criapi.RuntimeServiceClient, cId string) (*criapi.ContainerStatus, error) {

	if cId == "" {
		err := errdefs.InvalidInput("Container ID cannot be empty in GCCS")

		return nil, err
	}

	request := &criapi.ContainerStatusRequest{
		ContainerId: cId,
		Verbose:     false,
	}

	r, err := client.ContainerStatus(context.Background(), request)

	if err != nil {

		return nil, err
	}

	return r.Status, nil
}

// GetContainersForSandbox 获取容器
func GetContainersForSandbox(ctx context.Context, client criapi.RuntimeServiceClient, psId string) ([]*criapi.Container, error) {

	filter := &criapi.ContainerFilter{}
	filter.PodSandboxId = psId
	request := &criapi.ListContainersRequest{
		Filter: filter,
	}

	r, err := client.ListContainers(context.Background(), request)

	if err != nil {
		return nil, err
	}
	return r.Containers, nil
}

// GenerateContainerConfig 由node提供的pod配置，生成CRI需要的容器配置文件
func GenerateContainerConfig(ctx context.Context, container *v1.Container, pod *v1.Pod, imageRef, podVolRoot string,  attempt uint32) (*criapi.ContainerConfig, error) {

	config := &criapi.ContainerConfig{
		Metadata: &criapi.ContainerMetadata{
			Name:    container.Name,
			Attempt: attempt,
		},
		Image:       &criapi.ImageSpec{Image: imageRef},
		Command:     container.Command,
		Args:        container.Args,
		WorkingDir:  container.WorkingDir,
		//Envs:        createCtrEnvVars(container.Env),
		//Labels:      createCtrLabels(container, pod),
		//Annotations: createCtrAnnotations(container, pod),
		//Linux:       createCtrLinuxConfig(container, pod),
		LogPath:     fmt.Sprintf("%s-%d.log", container.Name, attempt),
		Stdin:       container.Stdin,
		StdinOnce:   container.StdinOnce,
		Tty:         container.TTY,
	}
	//mounts, err := createCtrMounts(ctx, container, pod, podVolRoot, rm)
	//if err != nil {
	//	return nil, err
	//}
	//config.Mounts = mounts
	return config, nil
}
