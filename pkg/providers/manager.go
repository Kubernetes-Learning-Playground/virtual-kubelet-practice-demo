package providers

import (
	"k8s.io/apimachinery/pkg/types"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// TODO: 使用PodManager管理pod状态

// PodManager pod管理器，用于存储node中的pod与其容器组状态
type PodManager struct {
	podStatus map[types.UID]PodStatus
	samplePodStatus  map[types.UID]PodStatus
}

func NewPodManager() *PodManager {
	return &PodManager{
		podStatus: map[types.UID]PodStatus{},
		samplePodStatus: map[types.UID]PodStatus{},
	}
}

func (pm *PodManager) getPodStatus() map[types.UID]PodStatus {
	return pm.podStatus
}

// PodStatus 单个pod的状态记录
type PodStatus struct {
	id string
	// containers 储存pod中容器组的状态
	containers map[string]*criapi.ContainerStatus
	// status pod的状态
	status *criapi.PodSandboxStatus
}
