package remote

import (
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// CRIContainer CRI服务端
type CRIContainer struct {
	RuntimeService v1alpha2.RuntimeServiceClient
	ImageService   v1alpha2.ImageServiceClient
}

func NewRemoteCRIContainer(runtimeService v1alpha2.RuntimeServiceClient, imageService v1alpha2.ImageServiceClient) *CRIContainer {
	return &CRIContainer{RuntimeService: runtimeService, ImageService: imageService}
}
