package providers

import (
	"bytes"
	"context"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	v1 "k8s.io/api/core/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
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
func (cc *ContainerCmd) Run() (string, string, error) {
	// 设置输出
	var stdout, stderr bytes.Buffer
	cc.Cmd.Stdout = &stdout
	cc.Cmd.Stderr = &stderr
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
	return string(stdout.Bytes()), string(stderr.Bytes()),err
}

func (c *CriProvider) createSamplePod(_ context.Context, pod *v1.Pod) error {
	// 1. 封装为ContainerCmd对象
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
			Cmd:           cmd,
			ContainerName: c.Name,
		})

	}

	// 2. 创建pod状态
	c.PodManager.samplePodStatus[pod.UID] = PodStatus{
		id: string(pod.UID),
		status: &criapi.PodSandboxStatus{
			Metadata: &criapi.PodSandboxMetadata{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Uid:       string(pod.UID),
			},
			Id:        string(pod.UID),
			State:     criapi.PodSandboxState_SANDBOX_READY,
			CreatedAt: time.Now().Unix(),
		},
		containers: map[string]*criapi.ContainerStatus{},
	}
	// 通知去更新状态
	c.notifyC <- struct{}{}

	for _, cmd := range cmds {
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName] = &criapi.ContainerStatus{
			Metadata: &criapi.ContainerMetadata{
				Name: cmd.ContainerName,
			},
			Id:        string(pod.UID) + cmd.ContainerName,
			CreatedAt: time.Now().Unix(),
			StartedAt: time.Now().Add(time.Second * 3).Unix(),
			State:     criapi.ContainerState_CONTAINER_CREATED,
			Message: "Creating",
		}

		c.notifyC <- struct{}{}

		// 修改容器状态为 running
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].State = criapi.ContainerState_CONTAINER_RUNNING
		c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Message = "Running"
		c.notifyC <- struct{}{}
		// 执行命令
		cmd := cmd
		go func() {
			outMessage, errMessage, err := cmd.Run()
			// 执行完毕，修改对应的状态
			// FIXME: 可封装为一个方法
			if err != nil {
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].State = criapi.ContainerState_CONTAINER_EXITED
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Reason = "Error"
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Message = errMessage
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].ExitCode = -9999
			} else {
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].State = criapi.ContainerState_CONTAINER_EXITED
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Reason = "Completed"
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].Message = outMessage
				c.PodManager.samplePodStatus[pod.UID].containers[cmd.ContainerName].ExitCode = 0
			}
			c.notifyC <- struct{}{}
		}()


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
	delete(c.PodManager.samplePodStatus, pod.UID)
	c.notifyStatus(pod)
	return nil
}
