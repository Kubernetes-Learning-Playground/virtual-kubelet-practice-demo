package providers

import (
	"context"
	"fmt"
	"github.com/practice/virtual-kubelet-practice/pkg/remote"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"os"
	"path/filepath"
	"time"

	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const PodLogRoot = "/var/log/vk-cri/"
const PodVolRoot = "/run/vk-cri/volumes/"
const PodLogRootPerms = 0755
const PodVolRootPerms = 0755

// CriProvider 对象
type CriProvider struct {
	// options 配置
	options *common.ProviderOption
	// cri客户端，包含runtimeService imageService
	remoteCRI *RemoteCRIContainer
	// provider需要存储该节点下的所有pod
	podStatus map[types.UID]CRIPod

	podLogRoot   string
	podVolRoot   string
	nodeName     string
	notifyStatus func(*v1.Pod)
}

// CRIPod 单个pod的状态记录
type CRIPod struct {
	id string
	// containers 储存pod中容器组的状态
	containers map[string]*criapi.ContainerStatus
	// status pod的状态
	status *criapi.PodSandboxStatus
}

// 是否实现下列两种接口，这是vk组件必须实现的两个接口。
var _ node.PodLifecycleHandler = &CriProvider{}
var _ node.PodNotifier = &CriProvider{}

func NewCriProvider(options *common.ProviderOption, criClient *RemoteCRIContainer) *CriProvider {
	c := &CriProvider{
		options:    options,
		remoteCRI:  criClient,
		podLogRoot: PodLogRoot,
		podVolRoot: PodVolRoot,
		podStatus:  map[types.UID]CRIPod{},
		nodeName:   "mynode",
	}

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
func (c *CriProvider) NotifyPods(ctx context.Context, f func(*v1.Pod)) {

	c.notifyStatus = f
	go c.statusLoop(ctx)
}

func (c *CriProvider) statusLoop(ctx context.Context) {
	t := time.NewTimer(5 * time.Second)
	if !t.Stop() {
		<-t.C
	}

	for {
		t.Reset(5 * time.Second)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		if err := c.notifyPodStatuses(ctx); err != nil {
			//log.G(ctx).WithError(err).Error("Error updating node statuses")
		}
	}
}

func (c *CriProvider) notifyPodStatuses(ctx context.Context) error {
	ls, err := c.GetPods(ctx)
	if err != nil {
		return err
	}

	for _, pod := range ls {
		c.notifyStatus(pod)
	}

	return nil
}

// CreatePod 创建pod的业务逻辑
func (c *CriProvider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("接收到来自k8s-apiserver的创建pod请求。")
	klog.Info("在此节点上，可以自定义加入业务逻辑。ex: 放入redis or etcd 或是放入数据库等")

	err := c.createPod(context.Background(), pod)

	return err

}

// UpdatePod 更新pod的业务逻辑
func (c *CriProvider) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("更新pod请求。")
	return nil
}

// DeletePod 删除pod的业务逻辑
func (c *CriProvider) DeletePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("pod被删除，名称是", pod.Name)
	err := c.deletePod(context.Background(), pod)
	return err
}

// 获取pod接口
func (c *CriProvider) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	klog.Infof("获取pod信息: %s, %s", namespace, name)
	pod, err := c.getPod(context.Background(), namespace, name)
	return pod, err
}

func (c *CriProvider) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	klog.Infof("获取pod状态status: %s, %s", name, namespace)
	pod, err := c.getPodStatus(context.Background(), namespace, name)
	return pod, err
}

func (c *CriProvider) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	var pods []*v1.Pod

	err := c.refreshNodeState(context.Background())
	if err != nil {
		return nil, err
	}

	for _, ps := range c.podStatus {
		pods = append(pods, createPodSpecFromCRI(&ps, c.nodeName))
	}

	return pods, nil
}

func (c *CriProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	fmt.Println("获取POD日志")
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
	node.Status.DaemonEndpoints = nodeDaemonEndpoints(c.options.DaemonEndpointPort)
	node.Status.NodeInfo.OperatingSystem = c.options.OperatingSystem
}

func (c *CriProvider) createPod(ctx context.Context, pod *v1.Pod) error {

	var attempt uint32 // TODO: Track attempts. Currently always 0
	logPath := filepath.Join(c.podLogRoot, string(pod.UID))
	volPath := filepath.Join(c.podVolRoot, string(pod.UID))
	err := c.refreshNodeState(context.Background())
	if err != nil {
		return err
	}
	pConfig, err := remote.GeneratePodSandboxConfig(context.Background(), pod, logPath, attempt)
	if err != nil {
		return err
	}
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
		// TODO: Is there a race here?
		pId, err = remote.RunPodSandbox(ctx, c.remoteCRI.RuntimeService, pConfig)
		if err != nil {
			return err
		}
	} else {
		pId = existing.status.Metadata.Uid
	}

	for _, cs := range pod.Spec.Containers {
		imageRef, err := remote.PullImage(context.Background(), c.remoteCRI.ImageService, cs.Image)
		if err != nil {
			return err
		}
		//log.G(ctx).Debugf("Creating container %s", c.Name)
		//cConfig, err := remote.GenerateContainerConfig(ctx, &cs, pod, imageRef, volPath, c.resourceManager, attempt)
		cConfig, err := remote.GenerateContainerConfig(context.Background(), &cs, pod, imageRef, volPath, attempt)
		if err != nil {
			return err
		}
		cId, err := remote.CreateContainer(context.Background(), c.remoteCRI.RuntimeService, cConfig, pConfig, pId)
		if err != nil {
			return err
		}
		//log.G(ctx).Debugf("Starting container %s", cs.Name)
		err = remote.StartContainer(context.Background(), c.remoteCRI.RuntimeService, cId)
	}

	return err
}

func (c *CriProvider) deletePod(ctx context.Context, pod *v1.Pod) error {
	//log.G(ctx).Debugf("receive DeletePod %q", pod.Name)

	err := c.refreshNodeState(context.Background())
	if err != nil {
		return err
	}

	ps, ok := c.podStatus[pod.UID]
	if !ok {
		return errdefs.NotFoundf("Pod %s not found", pod.UID)
	}

	// TODO: Check pod status for running state
	err = remote.StopPodSandbox(context.Background(), c.remoteCRI.RuntimeService, ps.status.Id)
	if err != nil {
		// Note the error, but shouldn't prevent us trying to delete
		//log.G(ctx).Debug(err)
	}

	// Remove any emptyDir volumes
	// TODO: Is there other cleanup that needs to happen here?
	err = os.RemoveAll(filepath.Join(c.podVolRoot, string(pod.UID)))
	if err != nil {
		//log.G(ctx).Debug(err)
	}
	err = remote.RemovePodSandbox(context.Background(), c.remoteCRI.RuntimeService, ps.status.Id)

	c.notifyStatus(pod)
	return err
}

func (c *CriProvider) getPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {

	err := c.refreshNodeState(context.Background())
	if err != nil {
		return nil, err
	}

	pod := c.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("Pod %s in namespace %s could not be found on the node", name, namespace)
	}

	return createPodSpecFromCRI(pod, c.nodeName), nil
}

// Find a pod by name and namespace. Pods are indexed by UID
func (c *CriProvider) findPodByName(namespace, name string) *CRIPod {
	var found *CRIPod

	for _, pod := range c.podStatus {
		if pod.status.Metadata.Name == name && pod.status.Metadata.Namespace == namespace {
			found = &pod
			break
		}
	}
	return found
}

func (c *CriProvider) getPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	//log.G(ctx).Debugf("receive GetPodStatus %q", name)

	err := c.refreshNodeState(context.Background())
	if err != nil {
		return nil, err
	}

	pod := c.findPodByName(namespace, name)
	if pod == nil {
		return nil, errdefs.NotFoundf("pod %s in namespace %s could not be found on the node", name, namespace)
	}

	return createPodStatusFromCRI(pod), nil
}

// Build an internal representation of the state of the pods and containers on the node
// Call this at the start of every function that needs to read any pod or container state
func (c *CriProvider) refreshNodeState(ctx context.Context) (retErr error) {

	allPods, err := remote.GetPodSandboxes(context.Background(), c.remoteCRI.RuntimeService)
	if err != nil {
		return err
	}

	newStatus := make(map[types.UID]CRIPod)
	for _, pod := range allPods {
		psId := pod.Id

		pss, err := remote.GetPodSandboxStatus(context.Background(), c.remoteCRI.RuntimeService, psId)
		if err != nil {
			return err
		}

		containers, err := remote.GetContainersForSandbox(context.Background(), c.remoteCRI.RuntimeService, psId)
		if err != nil {
			return err
		}

		var css = make(map[string]*criapi.ContainerStatus)
		for _, cc := range containers {
			cstatus, err := remote.GetContainerCRIStatus(context.Background(), c.remoteCRI.RuntimeService, cc.Id)
			if err != nil {
				return err
			}
			css[cstatus.Metadata.Name] = cstatus
		}

		newStatus[types.UID(pss.Metadata.Uid)] = CRIPod{
			id:         pod.Id,
			status:     pss,
			containers: css,
		}
	}
	c.podStatus = newStatus
	return nil
}

// Creates a Pod spec from data obtained through CRI
func createPodSpecFromCRI(p *CRIPod, nodeName string) *v1.Pod {
	cSpecs, _ := createContainerSpecsFromCRI(p.containers)

	// TODO: Fill out more fields here
	podSpec := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.status.Metadata.Name,
			Namespace: p.status.Metadata.Namespace,
			// ClusterName:       TODO: What is this??
			UID:               types.UID(p.status.Metadata.Uid),
			CreationTimestamp: metav1.NewTime(time.Unix(0, p.status.CreatedAt)),
		},
		Spec: v1.PodSpec{
			NodeName:   nodeName,
			Volumes:    []v1.Volume{},
			Containers: cSpecs,
		},
		Status: *createPodStatusFromCRI(p),
	}

	return &podSpec
}

// Converts CRI container spec to Container spec
func createContainerSpecsFromCRI(containerMap map[string]*criapi.ContainerStatus) ([]v1.Container, []v1.ContainerStatus) {
	containers := make([]v1.Container, 0, len(containerMap))
	containerStatuses := make([]v1.ContainerStatus, 0, len(containerMap))
	for _, c := range containerMap {
		// TODO: Fill out more fields
		container := v1.Container{
			Name:  c.Metadata.Name,
			Image: c.Image.Image,
			//Command:    Command is buried in the Info JSON,
		}
		containers = append(containers, container)
		// TODO: Fill out more fields
		containerStatus := v1.ContainerStatus{
			Name:        c.Metadata.Name,
			Image:       c.Image.Image,
			ImageID:     c.ImageRef,
			ContainerID: c.Id,
			Ready:       c.State == criapi.ContainerState_CONTAINER_RUNNING,
			State:       *createContainerStateFromCRI(c.State, c),
			// LastTerminationState:
			// RestartCount:
		}

		containerStatuses = append(containerStatuses, containerStatus)
	}
	return containers, containerStatuses
}

func createContainerStateFromCRI(state criapi.ContainerState, status *criapi.ContainerStatus) *v1.ContainerState {
	var result *v1.ContainerState
	switch state {
	case criapi.ContainerState_CONTAINER_UNKNOWN:
		fallthrough
	case criapi.ContainerState_CONTAINER_CREATED:
		result = &v1.ContainerState{
			Waiting: &v1.ContainerStateWaiting{
				Reason:  status.Reason,
				Message: status.Message,
			},
		}
	case criapi.ContainerState_CONTAINER_RUNNING:
		result = &v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.NewTime(time.Unix(0, status.StartedAt)),
			},
		}
	case criapi.ContainerState_CONTAINER_EXITED:
		result = &v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode:   status.ExitCode,
				Reason:     status.Reason,
				Message:    status.Message,
				StartedAt:  metav1.NewTime(time.Unix(0, status.StartedAt)),
				FinishedAt: metav1.NewTime(time.Unix(0, status.FinishedAt)),
			},
		}
	}
	return result
}

// Converts CRI pod status to a PodStatus
func createPodStatusFromCRI(p *CRIPod) *v1.PodStatus {
	_, cStatuses := createContainerSpecsFromCRI(p.containers)

	// TODO: How to determine PodSucceeded and PodFailed?
	phase := v1.PodPending
	if p.status.State == criapi.PodSandboxState_SANDBOX_READY {
		phase = v1.PodRunning
	}
	startTime := metav1.NewTime(time.Unix(0, p.status.CreatedAt))
	return &v1.PodStatus{
		Phase:             phase,
		Conditions:        []v1.PodCondition{},
		Message:           "",
		Reason:            "",
		HostIP:            "",
		PodIP:             p.status.Network.Ip,
		StartTime:         &startTime,
		ContainerStatuses: cStatuses,
	}
}