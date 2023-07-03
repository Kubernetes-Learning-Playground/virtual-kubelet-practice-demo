package common

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"log"
	"time"
)

const CriAddr = "unix:///run/containerd/containerd.sock"

var (
	R v1alpha2.RuntimeServiceClient
	I v1alpha2.ImageServiceClient
)

func init() {
	R = NewRuntimeService()
	I = NewImageService()
}

func NewRuntimeService() v1alpha2.RuntimeServiceClient {
	return v1alpha2.NewRuntimeServiceClient(grpcClient)
}

// TODO: 拉镜像查镜像都使用

func NewImageService() v1alpha2.ImageServiceClient {
	return v1alpha2.NewImageServiceClient(grpcClient)
}

var grpcClient *grpc.ClientConn // grpc连接

func InitClient() {
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := grpc.DialContext(ctx, CriAddr, grpcOpts...)
	if err != nil {
		log.Fatalln(err)
	}

	grpcClient = conn
}
