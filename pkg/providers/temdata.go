package providers

import (
	"golanglearning/new_project/virtual-kubelet-practice/pkg/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

//临时存了一个 集合
// 1、数据库 2、直接调用containerd 、cri-o 3、另外一个 k8s 集群交互，监听
// 模拟存储一个pod结合
var TempPods []*v1.Pod


// createPod 存入pod
func createPod(pod *v1.Pod)  {
	if pod.Spec.NodeName == common.NodeName {
		TempPods = append(TempPods, pod)
	}
}




func init() {
	TempPods = make([]*v1.Pod, 0)
}

// setPodsStatus 临时使用，设置pod状态为Running
func setPodsStatus( )  {
	//start:=true
	for i, _ := range TempPods {
		TempPods[i].Status.Phase = v1.PodRunning
		if len(TempPods[i].Status.ContainerStatuses) < len(TempPods[i].Spec.Containers) {
			for _, c := range TempPods[i].Spec.Containers {
				TempPods[i].Status.ContainerStatuses = append(TempPods[i].Status.ContainerStatuses,
					v1.ContainerStatus{
						Name: c.Name,
						Image: c.Image,
						Ready: true,
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{
								StartedAt: metav1.NewTime(time.Now()),
							},
						},
					},
				)
			}
		}

	}
}



