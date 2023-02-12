## 对virtual-kubelet的初探demo
项目背景：目前"边缘云"与"混合云"的架构概念逐渐成为未来云计算的主要部署型态与方向，因此相关的技术也随之兴起。Virtual-Kubelet便是其中之一。
Virtual-Kubelet是基于Kubelet的典型特性实现，向上伪装成Kubelet，从而模拟出Node对象，对接Kubernetes的原生资源对象；向下提供API，可对接其他资源管理平台提供的Provider。

本项目基于Virtual-Kubelet的基础上，运行简易的自定义kubelet，并实现上报pod信息等功能。
(由于是初探，功能完全不完善，没有任何业务逻辑，仅仅是启动项目而已，供自己学习所用)

virtual-kubelet项目图提供(项目地址:https://github.com/virtual-kubelet/virtual-kubelet)


### 项目启动步骤
机器a为主节点，上有k8s集群，机器b为边缘节点，需要使用virtual-kubelet加入集群中。
1. 需要先准备两台连通网的机器，其中一台需要先安装kubeadm k8s v1.22.0(机器a)。
2. 先把机器b安装kubectl命令行工具，并修改.kube/config，使其可以连通机器a的集群。
如下：机器VM-0-8-centos上没有k8s，但直接连上VM-0-16-centos的机器上。
```
[root@VM-0-8-centos ~]# kubectl get node
NAME              STATUS     ROLES                  AGE     VERSION
vm-0-16-centos    Ready      control-plane,master   80d     v1.22.3
```
3. 机器b上安装containerd与cri工具(crictl)。

4. 把.kube/config拷贝一份到项目根目录的config中
```
➜  config git:(sidecar_fix) ✗ ls
config.yaml
➜  config git:(sidecar_fix) ✗ pwd
/xxxxxx......./virtual-kubelet-practice/config

```
5. 在机器b上执行 go run main.go
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
6. 可以在主节点(机器a)上执行 deployment
```
[root@VM-0-8-centos test]# kubectl get pods | grep critest
critest-857b8c5576-d5dph               1/1     Running       0                27s
```
