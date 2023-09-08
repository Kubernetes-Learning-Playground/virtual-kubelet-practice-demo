package providers

import (
	"context"
	"io"

	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// CreatePod 创建pod
func (c *CriProvider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("接收到来自k8s-apiserver的创建pod请求。")
	klog.Info("在此节点上，可以自定义加入业务逻辑。ex: 放入redis or etcd 或是放入数据库等")
	// 使用annotation区分不同pod功能
	if pod.Annotations != nil && pod.Annotations["type"] == "bash" {
		return c.createSamplePod(ctx, pod)
	}

	return c.createPod(ctx, pod)
}

// UpdatePod 更新pod
func (c *CriProvider) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("更新pod请求。")
	return nil
}

// DeletePod 删除pod
func (c *CriProvider) DeletePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("pod被删除，名称是", pod.Name)
	if pod.Annotations != nil && pod.Annotations["type"] == "bash" {
		return c.deleteSamplePod(ctx, pod)
	}
	return c.deletePod(ctx, pod)
}

// GetPod 获取pod
func (c *CriProvider) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	klog.Infof("获取name: %s namespace: %s ,获取pod信息", name, namespace)
	pod, err := c.getPod(ctx, namespace, name)
	return pod, err
}

// GetPodStatus 获取pod状态
func (c *CriProvider) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	klog.Infof("获取name: %s namespace: %s ,pod状态status", name, namespace)
	pod, err := c.getPodStatus(ctx, namespace, name)
	return pod, err
}

// GetPods 获取pod列表
func (c *CriProvider) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	klog.Infof("获取pod列表")
	return c.getPods(ctx)
}

// GetContainerLogs 获取容器日志
func (c *CriProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	klog.Infof("获取pod name: %s namespace: %s container name: %s 日志", podName, namespace, containerName)
	return nil, nil
}

// RunInContainer 执行pod中的容器逻辑
func (c *CriProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return nil
}

// ConfigureNode 初始化自定义node节点信息
func (c *CriProvider) ConfigureNode(ctx context.Context, node *v1.Node) {
	node.Status.Capacity = nodeCapacity(c.options.ResourceCPU, c.options.ResourceMemory, c.options.MaxPod)
	node.Status.Conditions = nodeConditions()
	node.Status.Addresses = nodeAddresses(c.options.InternalIp)
	node.Status.DaemonEndpoints = nodeDaemonEndpoints(int(c.options.DaemonEndpointPort))
	node.Status.NodeInfo.OperatingSystem = c.options.OperatingSystem
}
