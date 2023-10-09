## virtual-kubelet 初探 demo
项目背景：目前"边缘云"与"混合云"的架构概念逐渐成为未来云计算的主要部署型态与方向，因此相关的技术也随之兴起。virtual-kubelet 便是其中之一。
virtual-kubelet 是基于 kubelet 的典型特性实现，向上伪装成 kubelet ，从而模拟出 Node 对象，对接 Kubernetes 的原生资源对象；向下提供 API，可对接其他资源管理平台(CRI、公有云资源、自定义扩展等)提供的 Provider 。

### 项目功能
1. 可启动 Pod 中的业务容器(前提是依赖 containerd 容器运行时，需要事先安装) 
2. 可执行 bash 脚本

可参考 yaml，[执行容器功能](./test/pod.yaml) [执行bash功能](./test/pod1.yaml)

virtual-kubelet 项目图提供 [virtual-kubelet项目地址](https://github.com/virtual-kubelet/virtual-kubelet)
![](https://github.com/googs1025/virtual-kubelet-practice-demo/blob/main/image/diagram.svg?raw=true)

### virtual-kubelet 接口实现
**virtual kubelet** 提供一个插件式的 **provider** 接口，让开发者可以自定义实现传统 kubelet 的功能。

自定义的 provider 可以用自己的配置文件和环境参数。
自定义的 provider 必须提供以下功能：
- 提供 pod、容器、资源的生命周期管理的功能
- 符合 virtual kubelet 提供的 API
```go
// PodLifecycleHandler是被PodController来调用，来管理被分配到node上的pod。
type PodLifecycleHandler interface {
    // CreatePod takes a Kubernetes Pod and deploys it within the provider.
    CreatePod(ctx context.Context, pod *corev1.Pod) error

    // UpdatePod takes a Kubernetes Pod and updates it within the provider.
    UpdatePod(ctx context.Context, pod *corev1.Pod) error

    // DeletePod takes a Kubernetes Pod and deletes it from the provider.
    DeletePod(ctx context.Context, pod *corev1.Pod) error

    // GetPod retrieves a pod by name from the provider (can be cached).
    // The Pod returned is expected to be immutable, and may be accessed
    // concurrently outside of the calling goroutine. Therefore it is recommended
    // to return a version after DeepCopy.
    GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)

    // GetPodStatus retrieves the status of a pod by name from the provider.
    // The PodStatus returned is expected to be immutable, and may be accessed
    // concurrently outside of the calling goroutine. Therefore it is recommended
    // to return a version after DeepCopy.
    GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error)

    // GetPods retrieves a list of all pods running on the provider (can be cached).
    // The Pods returned are expected to be immutable, and may be accessed
    // concurrently outside of the calling goroutine. Therefore it is recommended
    // to return a version after DeepCopy.
    GetPods(context.Context) ([]*corev1.Pod, error)
}

// PodNotifier是可选实现，该接口主要用来通知virtual kubelet的pod状态变化。
// 如果没有实现该接口，virtual-kubelet会定期检查所有pod的状态。
// PodNotifier is used as an extension to PodLifecycleHandler to support async updates of pod statuses.
type PodNotifier interface {
    // NotifyPods instructs the notifier to call the passed in function when
    // the pod status changes. It should be called when a pod's status changes.
    //
    // The provided pointer to a Pod is guaranteed to be used in a read-only
    // fashion. The provided pod's PodStatus should be up to date when
    // this function is called.
    //
    // NotifyPods must not block the caller since it is only used to register the callback.
    // The callback passed into `NotifyPods` may block when called.
    NotifyPods(context.Context, func(*corev1.Pod))
}

```

### 项目启动步骤
机器 a 为主节点，上有 k8s 集群，机器 b 为边缘节点，需要使用 virtual-kubelet 加入集群中。
1. 需要先准备两台连通网的机器，其中一台需要先安装 kubeadm k8s v1.22.0(机器a)。
2. 先把机器 b 安装 kubectl 命令行工具，并修改 .kube/config，使其可以连通机器 a 的集群。
如下：机器 VM-0-8-centos 上没有 k8s，但直接连上 VM-0-16-centos 的机器上。
```
[root@VM-0-8-centos ~]# kubectl get node
NAME              STATUS     ROLES                  AGE     VERSION
vm-0-16-centos    Ready      control-plane,master   80d     v1.22.3
```
3. 机器 b 上安装 containerd 与 cri 工具(crictl)。

4. 把 .kube/config 拷贝一份到项目根目录的 config 中
```
➜  config git:(sidecar_fix) ✗ ls
config.yaml
➜  config git:(sidecar_fix) ✗ pwd
/xxxxxx......./virtual-kubelet-practice/config

```
5. 在机器 b 上执行 go run main.go
```
go run main.go --provider mynode --kubeconfig ./config/config.yaml --nodename edgenode  # provider nodename等参数 可以在./pkg/common中设置，
```
执行成功如下所示：
```
[root@VM-0-8-centos virtual-kubelet-practice]# go run main.go --provider mynode --kubeconfig ./config/config.yaml --nodename edgenode
ERRO[0000] TLS certificates not provided, not setting up pod http server  caPath= certPath= keyPath= node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
INFO[0000] Initialized                                   node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
INFO[0000] Pod cache in-sync                             node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
INFO[0000] starting workers                              node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
INFO[0000] started workers                               node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
获取POD详细
收到创建POD的信息,然后我们在这里自己创建POD，或者干点你喜欢的事。哪怕插入点数据到数据库都行
INFO[0000] Created pod in provider
INFO[0000] Event(v1.ObjectReference{Kind:"Pod", Namespace:"default", Name:"critest-857b8c5576-b4x28", UID:"785f2adb-62a4-481d-821b-9d6fec6f4178", APIVersion:"v1", ResourceVersion:"17604070", FieldPath:""}): type: 'Normal' reason: 'ProviderCreateSuccess' Create pod in provider successfully  node=edgenode operatingSystem=Linux provider=mynode watchedNamespace=
```
6. 可以在主节点(机器 a)上执行 deployment
```
[root@VM-0-8-centos test]# kubectl get pods | grep critest
critest-857b8c5576-d5dph               1/1     Running       0                27s
```

#### 二进制部署
- 注：本项目仅支持二进制部署(kubelet 本身也是二进制部署)
```bash
[root@VM-0-8-centos virtual-kubelet-practice]# make build
go build -a -o virtual-kubelet main.go
[root@VM-0-8-centos virtual-kubelet-practice]# make run
./virtual-kubelet --provider example-provider --kubeconfig ./config/config.yaml --nodename mynode
ERRO[0000] TLS certificates not provided, not setting up pod http server  caPath= certPath= keyPath= node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
INFO[0000] Initialized                                   node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
INFO[0000] Pod cache in-sync                             node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
I1009 23:18:05.824749   18681 provider.go:59] 获取pod列表
INFO[0000] starting workers                              node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
INFO[0000] started workers                               node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
I1009 23:18:05.830693   18681 provider.go:45] 获取name: virtual-kubelet-pod-test-bash namespace: default ,获取pod信息
I1009 23:18:05.832134   18681 provider.go:26] 更新pod请求。
INFO[0000] Updated pod in provider
INFO[0000] Event(v1.ObjectReference{Kind:"Pod", Namespace:"default", Name:"virtual-kubelet-pod-test-bash", UID:"01da63ed-72ec-4839-a562-752d04b9225c", APIVersion:"v1", ResourceVersion:"73014532", FieldPath:""}): type: 'Normal' reason: 'ProviderUpdateSuccess' Update pod in provider successfully  node=mynode operatingSystem=Linux provider=example-provider watchedNamespace=
```

RoadMap:
1. 对接 etcd or redis，让在边缘节点的请求能得到缓存记录
2. 参考阿里 腾讯 华为的 provider 加上一些自己的业务逻辑。