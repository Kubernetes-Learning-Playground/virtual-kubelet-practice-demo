package providers

import (
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"runtime"
	"strconv"
)

// nodeDaemonEndpoints 返回节点端口
func nodeDaemonEndpoints(port int) v1.NodeDaemonEndpoints {
	return v1.NodeDaemonEndpoints{
		KubeletEndpoint: v1.DaemonEndpoint{
			Port: int32(port),
		},
	}
}

// nodeAddresses 获取node 内部IP
func nodeAddresses(internalIP string) []v1.NodeAddress {
	return []v1.NodeAddress{
		{
			Type:    "InternalIP",
			Address: internalIP, // 需要改
		},
	}
}

// nodeConditions 节点状态集合
func nodeConditions() []v1.NodeCondition {
	// TODO: Make this configurable
	return []v1.NodeCondition{
		{
			Type:               "Ready",
			Status:             v1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "virtual-kubelet is ready.",
		},
		{
			Type:               "OutOfDisk",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientDisk",
			Message:            "virtual-kubelet has sufficient disk space available",
		},
		{
			Type:               "MemoryPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "virtual-kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "virtual-kubelet has no disk pressure",
		},
		{
			Type:               "NetworkUnavailable",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "RouteCreated",
			Message:            "RouteController created a route",
		},
	}

}

// nodeCapacity 节点资源信息如：CPU or 内存 or 最大承受pod数量
func nodeCapacity(resourceCPU, resourceMemory, maxPod string) v1.ResourceList {
	if resourceCPU == "" {
		resourceCPU = strconv.Itoa(runtime.NumCPU())
	}
	if resourceMemory == "" {
		resourceMemory = strconv.Itoa(1024 * 1024 * 1024 * 500)
	}
	if maxPod == "" {
		maxPod = "200"
	}

	return v1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(resourceCPU),
		corev1.ResourceMemory: resource.MustParse(resourceMemory),
		corev1.ResourcePods:   resource.MustParse(maxPod), //最多创建
	}
}
