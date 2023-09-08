package providers

import (
	"context"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	v1 "k8s.io/api/core/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"os"
	"os/exec"
	"time"
)

// ContainerCmd 针对每个容器的执行命令
type ContainerCmd struct {
	Cmd           *exec.Cmd `json:"cmd"`
	ContainerName string    `json:"container_name"`
	ExitCode      int       `json:"exit_code"`
	ExecError     error     `json:"exec_error"`
}

// Run 执行命令
func (cc *ContainerCmd) Run() {
	// 标准输出
	cc.Cmd.Stdout = os.Stdout
	cc.Cmd.Stderr = os.Stderr
	// 执行cmd
	err := cc.Cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			cc.ExitCode = exitCode
		} else {
			cc.ExitCode = -9999 //代表是其他错误
			cc.ExecError = err
		}
	}
}

func (c *CriProvider) createSamplePod(_ context.Context, pod *v1.Pod) error {
	cmds := make([]*ContainerCmd, 0)
	for _, c := range pod.Spec.Containers {
		if len(c.Command) == 0 {
			continue
		}
		args := make([]string, 0)
		if len(c.Command) > 1 {
			args = append(args, c.Command[1:]...)
		}
		args = append(args, c.Args...)
		cmd := exec.Command(c.Command[0], args...)
		cmds = append(cmds, &ContainerCmd{
			Cmd: cmd,
			ContainerName: c.Name,
		})

	}

	c.PodManager.samplePodStatus[pod.UID] = PodStatus{
		id: string(pod.UID),
		status: &criapi.PodSandboxStatus{
			Metadata: &criapi.PodSandboxMetadata{
				Name: pod.Name,
				Namespace: pod.Namespace,
				Uid: string(pod.UID),
			},
			Id: string(pod.UID),
			State: criapi.PodSandboxState_SANDBOX_NOTREADY,
			CreatedAt: time.Now().Unix(),
		},
		containers: map[string]*criapi.ContainerStatus{},
	}

	for _, cmd := range cmds {
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName] = &criapi.ContainerStatus{
			Metadata: &criapi.ContainerMetadata{
				Name: cmd.ContainerName,
			},
			Id: string(pod.UID) + cmd.ContainerName,
			CreatedAt: time.Now().Unix(),
			StartedAt: time.Now().Add(time.Second * 3).Unix(),
			State: criapi.ContainerState_CONTAINER_CREATED,
		}
		cmd.Run()
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].State = criapi.ContainerState_CONTAINER_RUNNING
		cmd.Cmd.Output()
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].State = criapi.ContainerState_CONTAINER_EXITED
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Reason = "Completed"
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].ExitCode = 0

	}

	c.notifyStatus(pod)
	return nil

}

func (c *CriProvider) deleteSamplePod(_ context.Context, pod *v1.Pod) error {
	ps, ok := c.PodManager.samplePodStatus[pod.UID]
	if !ok {
		return errdefs.NotFoundf("Pod %s not found", pod.UID)
	}

	ps.status.State = criapi.PodSandboxState_SANDBOX_NOTREADY
	for _, ss := range ps.containers {
		ss.State = criapi.ContainerState_CONTAINER_EXITED
	}
	c.notifyStatus(pod)
	return nil
}