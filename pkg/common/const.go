package common

type ProviderOption struct {
	ProviderName       string
	NodeName           string
	OperatingSystem    string
	DaemonEndpointPort int // 默认端口 10250
	InternalIp         string
	ResourceCPU        string
	ResourceMemory     string
	MaxPod             string
}
