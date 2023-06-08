package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

var (
	namespaceKey = "default"
	// 镜像
	imageName     = "docker.io/nginx:1.18-alpine"
	containerName = "test"
)

/*
	调用 ctr 启动容器，不能用在pod上，只能启动容器
*/

func main() {

	p := NewProvider()

	ctx := namespaces.WithNamespace(context.Background(), namespaceKey)

	i, err := p.GetImage(ctx, imageName)
	if err != nil {
		return
	} else if i.Name() == "" {
		// 拉取镜像
		err = p.PullImage(ctx, imageName)
		if err != nil {
			return
		}
		i, err = p.GetImage(ctx, imageName)
		if err != nil {
			return
		}
	}

	// 创建容器
	container, err := p.CreateContainer(ctx, containerName, "", i)
	if err != nil {
		return
	}
	fmt.Println("container id: ", container.ID())

}

const defaultAddress = "/run/containerd/containerd.sock"

type Provider struct {
	criClient *containerd.Client
}

func NewProvider() *Provider {
	c, err := NewCRIClient(defaultAddress)
	if err != nil {
		log.Fatalln("client init err: ", err)
	}
	return &Provider{criClient: c}
}

func NewCRIClient(address string) (*containerd.Client, error) {
	c, err := containerd.New(address)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *Provider) GetImage(ctx context.Context, imageName string) (containerd.Image, error) {
	im, err := p.criClient.GetImage(ctx, imageName)
	if err != nil {
		return nil, err
	}
	return im, nil
}

func (p *Provider) PullImage(ctx context.Context, imageName string) error {
	_, err := p.criClient.Pull(ctx, imageName)
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) CreateContainer(ctx context.Context, containerName, command string, image containerd.Image) (containerd.Container, error) {
	var container containerd.Container
	var err error
	if command != "" {
		container, err = p.criClient.NewContainer(ctx, containerName, containerd.WithImage(image), containerd.WithNewSnapshot(containerName, image),
			containerd.WithNewSpec(oci.WithImageConfig(image),
				oci.WithProcessArgs(strings.Split(command, " ")...)))
	} else {
		container, err = p.criClient.NewContainer(ctx, containerName, containerd.WithImage(image), containerd.WithNewSnapshot(containerName, image),
			containerd.WithNewSpec(oci.WithImageConfig(image)))
	}

	if err != nil {
		return nil, err
	}

	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, err
	}

	/// start the task
	if err := task.Start(ctx); err != nil {
		task.Delete(ctx)
		return nil, err
	}

	return container, nil
}
