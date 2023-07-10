package remote

import (
	"context"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	v1 "k8s.io/api/core/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// RunPodSandbox 执行PodSandbox请求
func RunPodSandbox(ctx context.Context, client criapi.RuntimeServiceClient, config *criapi.PodSandboxConfig) (string, error) {

	// 请求
	request := &criapi.RunPodSandboxRequest{Config: config}

	// 发送
	r, err := client.RunPodSandbox(context.Background(), request)
	if err != nil {
		return "", err
	}
	return r.PodSandboxId, nil
}

// StopPodSandbox 停止PodSandbox请求
func StopPodSandbox(ctx context.Context, client criapi.RuntimeServiceClient, id string) error {
	if id == "" {
		err := errdefs.InvalidInput("ID cannot be empty")
		return err
	}
	request := &criapi.StopPodSandboxRequest{PodSandboxId: id}
	_, err := client.StopPodSandbox(context.Background(), request)
	if err != nil {
		return err
	}

	return nil
}

// RemovePodSandbox 删除PodSandbox请求
func RemovePodSandbox(ctx context.Context, client criapi.RuntimeServiceClient, id string) error {

	if id == "" {
		err := errdefs.InvalidInput("ID cannot be empty")
		return err
	}
	request := &criapi.RemovePodSandboxRequest{PodSandboxId: id}

	_, err := client.RemovePodSandbox(context.Background(), request)

	if err != nil {
		return err
	}
	return nil
}

// GetPodSandboxes 获取PodSandboxes请求
func GetPodSandboxes(ctx context.Context, client criapi.RuntimeServiceClient) ([]*criapi.PodSandbox, error) {

	filter := &criapi.PodSandboxFilter{}
	request := &criapi.ListPodSandboxRequest{
		Filter: filter,
	}

	r, err := client.ListPodSandbox(context.Background(), request)

	if err != nil {
		return nil, err
	}
	return r.GetItems(), err
}

// GetPodSandboxStatus 获取 PodSandbox 状态
func GetPodSandboxStatus(ctx context.Context, client criapi.RuntimeServiceClient, psId string) (*criapi.PodSandboxStatus, error) {

	if psId == "" {
		err := errdefs.InvalidInput("Pod ID cannot be empty in GPSS")
		return nil, err
	}

	request := &criapi.PodSandboxStatusRequest{
		PodSandboxId: psId,
		Verbose:      false,
	}

	r, err := client.PodSandboxStatus(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return r.Status, nil
}

// GeneratePodSandboxConfig 从node给的pod配置生成CRI所需要的配置文件
func GeneratePodSandboxConfig(ctx context.Context, pod *v1.Pod, logDir string, attempt uint32) (*criapi.PodSandboxConfig, error) {
	podUID := string(pod.UID)
	config := &criapi.PodSandboxConfig{
		Metadata: &criapi.PodSandboxMetadata{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Uid:       podUID,
			Attempt:   attempt,
		},
		//Labels:       createPodLabels(pod),
		Annotations:  pod.Annotations,
		LogDirectory: logDir,
		//DnsConfig:    createPodDnsConfig(pod),
		//Hostname:     createPodHostname(pod),
		//PortMappings: createPortMappings(pod),
		//Linux:        createPodSandboxLinuxConfig(pod),
	}
	return config, nil
}
