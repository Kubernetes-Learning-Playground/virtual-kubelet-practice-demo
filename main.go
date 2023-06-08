package main

import (
	"context"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/providers"
	"github.com/sirupsen/logrus"
	cli "github.com/virtual-kubelet/node-cli"
	logruscli "github.com/virtual-kubelet/node-cli/logrus"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
	"os"
	"os/signal"
	"syscall"
)

var (
	providerName       string
	nodeName           string
	k8sVersion         string
	internalIp         string
	resourceCPU        string
	resourceMemory     string
	maxPod             string
	daemonEndpointPort int
	operatingSystem    string
)

func main() {
	// FIXME: 配置bug
	//flag.StringVar(&providerName, "providerName", "example-provider", "virtual-kubelet provider name")
	//flag.StringVar(&nodeName, "nodeName", "edgenode", "virtual-kubelet node name")
	//flag.StringVar(&k8sVersion, "k8sVersion", "v1.22.0", "virtual-kubelet k8s version")
	//flag.StringVar(&internalIp, "internalIp", "127.0.0.1", "virtual-kubelet internal ip")
	//flag.StringVar(&resourceCPU, "resourceCPU", "", "virtual-kubelet node cpu resource")
	//flag.StringVar(&resourceMemory, "resourceMemory", "", "virtual-kubelet node memory resource")
	//flag.StringVar(&maxPod, "maxPod", "", "virtual-kubelet node max pod number")
	//flag.IntVar(&daemonEndpointPort, "daemonEndpointPort", 10250, "virtual-kubelet node endpoint port")
	//flag.StringVar(&operatingSystem, "operatingSystem", "Linux", "virtual-kubelet node os")
	//flag.Parse()  // 不要解析，框架会解析

	common.InitClient()

	opt := common.ProviderOption{
		ProviderName:       "example-provider",
		NodeName:           "mynode",
		InternalIp:         "127.0.0.1",
		ResourceCPU:        "",
		ResourceMemory:     "",
		MaxPod:             "",
		DaemonEndpointPort: 10250,
		OperatingSystem:    "Linux",
	}

	//opt := common.ProviderOption{
	//	ProviderName:       providerName,
	//	NodeName:           nodeName,
	//	InternalIp:         internalIp,
	//	ResourceCPU:        resourceCPU,
	//	ResourceMemory:     resourceMemory,
	//	MaxPod:             maxPod,
	//	DaemonEndpointPort: daemonEndpointPort,
	//	OperatingSystem:    operatingSystem,
	//}

	_, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	ctx := cli.ContextWithCancelOnSignal(context.Background())
	logger := logrus.StandardLogger()

	log.L = logruslogger.FromLogrus(logrus.NewEntry(logger))
	logConfig := &logruscli.Config{LogLevel: "info"}

	node, err := cli.New(ctx,

		cli.WithProvider("example-provider", func(cfg provider.InitConfig) (provider.Provider, error) {
			return providers.NewCriProvider(&opt), nil
		}),
		cli.WithKubernetesNodeVersion("v1.22.0"),
		// Adds flags and parsing for using logrus as the configured logger
		cli.WithPersistentFlags(logConfig.FlagSet()),
		cli.WithPersistentPreRunCallback(func() error {
			return logruscli.Configure(logConfig, logger)
		}),
	)

	if err != nil {
		panic(err)
	}
	// Args can be specified here, or os.Args[1:] will be used.
	if err := node.Run(ctx); err != nil {
		panic(err)
	}
}
