package providers

import (
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// RemoteCRIContainer CRI服务端
type RemoteCRIContainer struct {
	RuntimeService v1alpha2.RuntimeServiceClient
	ImageService   v1alpha2.ImageServiceClient
}

func NewRemoteCRIContainer(runtimeService v1alpha2.RuntimeServiceClient, imageService v1alpha2.ImageServiceClient) *RemoteCRIContainer {
	return &RemoteCRIContainer{RuntimeService: runtimeService, ImageService: imageService}
}
