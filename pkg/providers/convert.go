package providers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"time"
)

// createPodSpecFromCRI 由CRI配置创建出pod对象
func createPodSpecFromCRI(p *PodStatus, nodeName string) *v1.Pod {
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

// createContainerSpecsFromCRI 由CRI配置创建出Container与ContainerStatus
func createContainerSpecsFromCRI(containerMap map[string]*criapi.ContainerStatus) ([]v1.Container, []v1.ContainerStatus) {
	containers := make([]v1.Container, 0, len(containerMap))
	containerStatuses := make([]v1.ContainerStatus, 0, len(containerMap))
	for _, c := range containerMap {
		// TODO: Fill out more fields
		container := v1.Container{
			Name:  c.Metadata.Name,
			Image: handleImage(c),

			//Command:    Command is buried in the Info JSON,
		}
		containers = append(containers, container)
		// TODO: Fill out more fields
		containerStatus := v1.ContainerStatus{
			Name:        c.Metadata.Name,
			Image:       handleImage(c),
			ImageID:     handleImageRef(c),
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

// createContainerStateFromCRI 转换为ContainerState
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

// createPodStatusFromCRI 由CRI配置创建出pod对象
func createPodStatusFromCRI(p *PodStatus) *v1.PodStatus {
	_, cStatuses := createContainerSpecsFromCRI(p.containers)

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
		PodIP:             handleNetworkIp(p),
		StartTime:         &startTime,
		ContainerStatuses: cStatuses,
	}
}

func handleNetworkIp(pp *PodStatus) (string) {
	if pp.status.Network == nil {
		return ""
	}
	return pp.status.Network.Ip
}

func handleImage(cc *criapi.ContainerStatus) string {
	if cc.Image == nil {
		return ""
	}
	return cc.Image.GetImage()
}

func handleImageRef(cc *criapi.ContainerStatus) string {
	if cc.Image == nil {
		return ""
	}
	return cc.GetImageRef()
}
