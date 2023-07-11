package common

import "github.com/virtual-kubelet/node-cli/provider"

// ProviderConfig provider 配置文件
type ProviderConfig struct {
	// NodeName 节点名
	NodeName string
	// OperatingSystem 启动节点的操作系统
	OperatingSystem string
	// DaemonEndpointPort 默认端口 10250
	DaemonEndpointPort int32
	// InternalIp 地址
	InternalIp string
	// ResourceCPU 节点cpu资源
	ResourceCPU string
	// ResourceMemory 节点内存
	ResourceMemory string
	// MaxPod 最大pod数
	MaxPod string
}

// SetupConfig 设置配置文件
func SetupConfig(cfg provider.InitConfig) *ProviderConfig {
	return &ProviderConfig{
		NodeName:           cfg.NodeName,
		OperatingSystem:    cfg.OperatingSystem,
		DaemonEndpointPort: cfg.DaemonPort,
		InternalIp:         cfg.InternalIP,
	}
}
