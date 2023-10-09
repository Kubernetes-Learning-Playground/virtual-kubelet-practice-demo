package providers

import (
	"context"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/remote"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	PodLogRoot      = "/var/log/vk-cri/"
	PodVolRoot      = "/run/vk-cri/volumes/"
	PodLogRootPerms = 0755
	PodVolRootPerms = 0755
)

// CriProvider 实现virtual-kubelet对象
type CriProvider struct {
	// options 配置
	options *common.ProviderConfig
	// cri客户端，包含runtimeService imageService
	remoteCRI *remote.CRIContainer
	// PodManager 管理pods状态管理
	PodManager *PodManager
	// podLogRoot 存放容器日志目录
	podLogRoot string
	// podVolRoot 存放容器挂载目录
	podVolRoot string
	// nodeName 节点名称，初始化时必须指定
	nodeName string
	// checkPeriod 检查定时周期
	checkPeriod int64
	notifyC     chan struct{}
	// 上报的回调方法，主要把本节点中的pod status放入工作队列
	notifyStatus func(*v1.Pod)

	// 模拟实现，
	// TODO: 发消息管理器，主要负责发送消息通知，是否发送消息，可以使用annotation标示发送
	// TODO: 数据库存储
}

// 是否实现下列两种接口，这是vk组件必须实现的两个接口。
var _ node.PodLifecycleHandler = &CriProvider{}
var _ node.PodNotifier = &CriProvider{}

func NewCriProvider(options *common.ProviderConfig, criClient *remote.CRIContainer) *CriProvider {

	c := &CriProvider{
		options:    options,
		remoteCRI:  criClient,
		podLogRoot: PodLogRoot,
		podVolRoot: PodVolRoot,
		PodManager: NewPodManager(),
		nodeName:   options.NodeName,
		notifyC:    make(chan struct{}),
	}
	// 初始化时先创建目录
	err := os.MkdirAll(c.podLogRoot, PodLogRootPerms)
	if err != nil {
		return nil
	}
	err = os.MkdirAll(c.podVolRoot, PodVolRootPerms)
	if err != nil {
		return nil
	}
	return c
}

// NotifyPods 异步更新pod的状态。
// 需要实现 node.PodNotifier 对象
func (c *CriProvider) NotifyPods(ctx context.Context, notifyStatus func(*v1.Pod)) {
	c.notifyStatus = notifyStatus
	go c.checkPodStatusLoop(ctx)
	go c.checkSamplePodStatusLoop()
}

// FIXME: 暂时使用此方式 通知
func (c *CriProvider) checkSamplePodStatusLoop() {
	for {
		select {
		case <-c.notifyC:
			var pods []*v1.Pod

			for _, ps := range c.PodManager.samplePodStatus {
				pods = append(pods, createPodSpecFromCRI(&ps, c.nodeName))
			}

			for _, pod := range pods {
				c.notifyStatus(pod)
			}
		}
	}

}

const defaultCheckPeriod = 5

// checkPodStatusLoop 定时检查pod状态
func (c *CriProvider) checkPodStatusLoop(ctx context.Context) {
	if c.checkPeriod <= 0 {
		c.checkPeriod = defaultCheckPeriod
	}
	t := time.NewTimer(time.Duration(c.checkPeriod))
	if !t.Stop() {
		<-t.C
	}
	// 重新计时执行
	for {
		t.Reset(time.Duration(c.checkPeriod) * time.Second)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		if err := c.notifyPodStatuses(ctx); err != nil {
			klog.Error("notifyPodStatuses err: ", err)
		}
	}
}

// notifyPodStatuses 获取pod，并通知node节点上报
func (c *CriProvider) notifyPodStatuses(ctx context.Context) error {
	pods, err := c.getPods(ctx)
	if err != nil {
		klog.Error("get pods err: ", err)
		return err
	}

	for _, pod := range pods {
		c.notifyStatus(pod)
	}

	return nil
}

// createPod 创建pod业务逻辑
func (c *CriProvider) createPod(ctx context.Context, pod *v1.Pod) error {

	var attempt uint32 // TODO: Track attempts. Currently always 0
	logPath := filepath.Join(c.podLogRoot, string(pod.UID))
	volPath := filepath.Join(c.podVolRoot, string(pod.UID))
	// 刷新node中状态
	err := c.refreshNodeState(ctx)
	if err != nil {
		klog.Error("refreshNodeState err: ", err)
		return err
	}
	// 生成pod sandbox配置文件
	pConfig, err := remote.GeneratePodSandboxConfig(ctx, pod, logPath, attempt)
	if err != nil {
		klog.Error("GeneratePodSandboxConfig err: ", err)
		return err
	}
	// 获取pod对象，用于判断是否创建过。
	existing := c.findPodByName(pod.Namespace, pod.Name)

	// TODO: Is re-using an existing sandbox with the UID the correct behavior?
	// TODO: Should delete the sandbox if container creation fails
	var pId string
	if existing == nil {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			return err
		}
		err = os.MkdirAll(volPath, 0755)
		if err != nil {
			return err
		}
		// 创建pod sandbox
		pId, err = remote.RunPodSandbox(ctx, c.remoteCRI.RuntimeService, pConfig)
		if err != nil {
			klog.Error("RunPodSandbox err: ", err)
			return err
		}
	} else {
		pId = existing.status.Metadata.Uid
	}

	klog.Infof("PodSandbox id %s", pId)

	// 执行创建容器相关的操作
	for _, cs := range pod.Spec.Containers {
		// 拉取镜像
		imageRef, err := remote.PullImage(ctx, c.remoteCRI.ImageService, cs.Image)
		if err != nil {
			klog.Error("PullImage err: ", err)
			return err
		}

		klog.Infof("Creating container %s", cs.Name)
		//cConfig, err := remote.GenerateContainerConfig(ctx, &cs, pod, imageRef, volPath, c.resourceManager, attempt)
		// 生成容器配置文件
		cConfig, err := remote.GenerateContainerConfig(ctx, &cs, pod, imageRef, volPath, attempt)
		if err != nil {
			klog.Error("GenerateContainerConfig err: ", err)
			return err
		}
		// 创建容器
		cId, err := remote.CreateContainer(ctx, c.remoteCRI.RuntimeService, cConfig, pConfig, pId)
		if err != nil {
			klog.Error("CreateContainer err: ", err)
			return err
		}

		klog.Infof("Starting container %s", cs.Name)
		// 运行容器
		err = remote.StartContainer(context.Background(), c.remoteCRI.RuntimeService, cId)
		if err != nil {
			klog.Error("StartContainer err: ", err)
			return err
		}
	}
	c.notifyStatus(pod)
	return err
}

// deletePod 删除pod业务逻辑
func (c *CriProvider) deletePod(ctx context.Context, pod *v1.Pod) error {
	klog.Infof("receive DeletePod %s", pod.Name)
	// 刷新node中的pod状态
	err := c.refreshNodeState(ctx)
	if err != nil {
		klog.Errorf("refreshNodeState err: %s", err)
		return err
	}

	ps, ok := c.PodManager.podStatus[pod.UID]
	if !ok {
		return errdefs.NotFoundf("Pod %s not found", pod.UID)
	}

	// TODO: Check pod status for running state
	// 停止pod sandbox
	err = remote.StopPodSandbox(ctx, c.remoteCRI.RuntimeService, ps.status.Id)
	if err != nil {
		// Note the error, but shouldn't prevent us trying to delete
		klog.Error("StopPodSandbox err: ", err)
	}

	// 删除日志文件
	err = os.RemoveAll(filepath.Join(c.podVolRoot, string(pod.UID)))
	if err != nil {
		klog.Error("Remove file err: ", err)
	}
	// 删除pod sandbox
	err = remote.RemovePodSandbox(ctx, c.remoteCRI.RuntimeService, ps.status.Id)
	if err != nil {
		klog.Error("RemovePodSandbox err: ", err)
		return err
	}
	c.notifyStatus(pod)
	return err
}

// getPod 获取pod
func (c *CriProvider) getPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	// 刷新node中pod状态
	err := c.refreshNodeState(ctx)
	if err != nil {
		return nil, err
	}
	// 查到pod
	pod := c.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("Pod %s in namespace %s could not be found on the node", name, namespace)
	}

	return createPodSpecFromCRI(pod, c.nodeName), nil
}

// getPod 获取pod列表
func (c *CriProvider) getPods(ctx context.Context) ([]*v1.Pod, error) {
	var pods []*v1.Pod
	// 刷新node中pod状态
	err := c.refreshNodeState(ctx)
	if err != nil {
		return nil, err
	}
	// 生成k8s中的pod对象
	for _, ps := range c.PodManager.podStatus {
		pods = append(pods, createPodSpecFromCRI(&ps, c.nodeName))
	}
	// 生成k8s中的pod对象
	for _, ps := range c.PodManager.samplePodStatus {
		pods = append(pods, createPodSpecFromCRI(&ps, c.nodeName))
	}
	return pods, nil
}

// findPodByName 从内存中获取pod状态
func (c *CriProvider) findPodByName(namespace, name string) *PodStatus {
	var found *PodStatus

	for _, pod := range c.PodManager.getSamplePodStatus() {
		if pod.status.Metadata.Name == name && pod.status.Metadata.Namespace == namespace {
			found = &pod
			break
		}
	}

	for _, pod := range c.PodManager.getPodStatus() {
		if pod.status.Metadata.Name == name && pod.status.Metadata.Namespace == namespace {
			found = &pod
			break
		}
	}
	return found
}

// getPodStatus 获取pod状态
func (c *CriProvider) getPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	//log.G(ctx).Debugf("receive GetPodStatus %q", name)
	// 刷新node中pod状态
	err := c.refreshNodeState(ctx)
	if err != nil {
		return nil, err
	}
	// 获取pod
	pod := c.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("pod %s in namespace %s could not be found on the node", name, namespace)
	}
	// 返回k8s中pod对象
	return createPodStatusFromCRI(pod), nil
}

// refreshNodeState 更新node中的pod状态
func (c *CriProvider) refreshNodeState(ctx context.Context) (retErr error) {
	// 获取pod sandbox
	allPods, err := remote.GetPodSandboxes(ctx, c.remoteCRI.RuntimeService)
	if err != nil {
		return err
	}

	newStatus := make(map[types.UID]PodStatus)
	for _, pod := range allPods {
		psId := pod.Id
		// 获取pod sandbox状态
		pss, err := remote.GetPodSandboxStatus(ctx, c.remoteCRI.RuntimeService, psId)
		if err != nil {
			return err
		}
		// 取得特定pod sandbox下的容器组
		containers, err := remote.GetContainersForSandbox(ctx, c.remoteCRI.RuntimeService, psId)
		if err != nil {
			return err
		}

		var css = make(map[string]*criapi.ContainerStatus)
		for _, cc := range containers {
			// 获取容器的状态
			cstatus, err := remote.GetContainerCRIStatus(context.Background(), c.remoteCRI.RuntimeService, cc.Id)
			if err != nil {
				return err
			}
			css[cstatus.Metadata.Name] = cstatus
		}

		newStatus[types.UID(pss.Metadata.Uid)] = PodStatus{
			id:         pod.Id,
			status:     pss,
			containers: css,
		}
	}
	c.PodManager.podStatus = newStatus
	return nil
}
