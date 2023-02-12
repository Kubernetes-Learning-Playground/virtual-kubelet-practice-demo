package providers

import (
	"context"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"time"
	"fmt"
)

// CriProvider 对象
type CriProvider struct {
	OS 				string
	EndpointPort 	int32 // 默认端口 10250
}


func NewCriProvider(OS string, endpoint int32) *CriProvider {
	return &CriProvider{OS: OS, EndpointPort: endpoint}
}

// 是否实现下列两种接口。
var _ node.PodLifecycleHandler = &CriProvider{}
var _ node.PodNotifier = &CriProvider{}


// NotifyPods 异步更新pod的状态。
// 需要实现 node.PodNotifier 对象
func (c CriProvider) NotifyPods(ctx context.Context, f func(*v1.Pod)) {
	go func() {
		for {
			time.Sleep(time.Second*3)
			setPodsStatus() //临时代码
			for _, pod := range TempPods {
				f(pod)
			}
		}
	}()
}


// CreatePod 创建pod的业务逻辑
func (c CriProvider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("接收到来自k8s-apiserver的创建pod请求。")
	klog.Info("在此节点上，可以自定义加入业务逻辑。ex: 放入redis or etcd 或是放入数据库等")
	createPod(pod)
	return nil
}

// UpdatePod 更新pod的业务逻辑
func (c CriProvider) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("更新pod请求。")
	return nil
}

// DeletePod 删除pod的业务逻辑
func (c CriProvider) DeletePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("pod被删除，名称是",pod.Name)
	return nil
}

// 获取pod接口

func (c CriProvider) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	klog.Infof("获取pod信息: ", namespace, name)
	return nil, nil
}

func (c CriProvider) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	klog.Infof("获取pod状态status: ", name, namespace)
	return nil, nil
}

func (c CriProvider) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	return nil, nil
}

func (c CriProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	fmt.Println("获取POD日志")
	return nil, nil
}

// RunInContainer 执行pod中的容器逻辑
func (c CriProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach api.AttachIO) error {
	return nil
}

// ConfigureNode 初始化自定义node节点信息
func (c CriProvider) ConfigureNode(ctx context.Context, node *v1.Node) {
	node.Status.Capacity = nodeCapacity()
	node.Status.Conditions = nodeConditions()
	node.Status.Addresses = nodeAddresses()
	node.Status.DaemonEndpoints = nodeDaemonEndpoints(c.EndpointPort)
	node.Status.NodeInfo.OperatingSystem = c.OS
}

