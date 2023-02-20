package providers

import (
	"context"
	"fmt"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	"log"
	"os"
	"time"
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

const CriAddr = "unix:///run/containerd/containerd.sock" //临时写死

var grpcClient  *grpc.ClientConn  // grpc连接

func InitClient()  {
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := grpc.DialContext(ctx, CriAddr, grpcOpts...)
	if err != nil {
		log.Fatalln(err)
	}

	grpcClient = conn
}

func NewRuntimeService() v1alpha2.RuntimeServiceClient  {
	return v1alpha2.NewRuntimeServiceClient(grpcClient)
}

func NewImageService() v1alpha2.ImageServiceClient{
	return v1alpha2.NewImageServiceClient(grpcClient)
}

// CreatePod 创建pod的业务逻辑
func (c CriProvider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	klog.Info("接收到来自k8s-apiserver的创建pod请求。")
	klog.Info("在此节点上，可以自定义加入业务逻辑。ex: 放入redis or etcd 或是放入数据库等")
	klog.Info(pod)
	klog.Info("name:", pod.Spec.Containers[0].Name)
	klog.Info("image:", pod.Spec.Containers[0].Image)
	klog.Info("command:", pod.Spec.Containers[0].Command)

	config1 := &v1alpha2.PodSandboxConfig{}
	err := YamlFile2Struct("./test/sandbox.yaml", config1)
	klog.Info(config1)
	if err != nil {
		klog.Error(err)
		return nil
	}
	config1.Metadata.Namespace = pod.Namespace
	config1.Metadata.Name = pod.Name
	config1.LogDirectory =  "/root/temp"
	//config1.PortMappings[0].ContainerPort = pod.Spec.Containers[0].Ports[0].ContainerPort


	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	req := &v1alpha2.RunPodSandboxRequest{
		Config: config1,
	}
	rsp, err := NewRuntimeService().RunPodSandbox(ctx, req)
	if err != nil {
		klog.Error(err)
		return nil
	}
	fmt.Println(rsp.PodSandboxId)


	podId := rsp.PodSandboxId

	config := &v1alpha2.ContainerConfig{}
	klog.Infof("aaa", config)
	err = YamlFile2Struct("./test/nginx.yaml", config)
	if err != nil {
		klog.Error(err)
	}
	config.Metadata.Name = pod.Spec.Containers[0].Name
	config.Image.Image = pod.Spec.Containers[0].Image
	config.Command = pod.Spec.Containers[0].Command


	ctx, cancel = context.WithTimeout(context.Background(),time.Second*10)
	defer cancel()

	//POD sandbox对应的配置对象
	pConfig := &v1alpha2.PodSandboxConfig{}
	klog.Infof("aaa", pConfig)
	err = YamlFile2Struct("./test/sandbox.yaml", pConfig)
	if err != nil  {
		klog.Error(err)
		return nil
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

	runtimeService := NewRuntimeService()
	rsp1, err := runtimeService.CreateContainer(ctx, req1)
	if err != nil {
		klog.Error(err)
	}
	// 启动容器
	sreq := &v1alpha2.StartContainerRequest{ContainerId: rsp1.ContainerId}

	_, err = runtimeService.StartContainer(ctx, sreq)
	if err != nil {
		klog.Error(err)
		return nil
	}

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


// YamlFile2Struct 读取文件内容 且反序列为struct
func YamlFile2Struct(path string,obj interface{}) error{
	b, err := GetFileContent(path)
	if err != nil {
		klog.Error("开启文件错误：", err)
		return err
	}
	err = yaml.Unmarshal(b, obj)
	if err != nil {
		klog.Error("解析yaml文件错误：", err)
		return err
	}
	return nil
}


// GetFileContent 文件读取函数
func GetFileContent(path string) ([]byte,error){
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}


