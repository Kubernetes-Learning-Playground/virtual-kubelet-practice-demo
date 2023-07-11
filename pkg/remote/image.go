package remote

import (
	"context"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// PullImage 拉取镜像请求
func PullImage(ctx context.Context, client criapi.ImageServiceClient, image string) (string, error) {

	// 请求
	request := &criapi.PullImageRequest{
		Image: &criapi.ImageSpec{
			Image: image,
		},
	}

	// 发送请求
	r, err := client.PullImage(ctx, request)
	if err != nil {
		return "", err
	}

	return r.ImageRef, nil
}
