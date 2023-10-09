package main

import (
	"context"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/providers"
	"github.com/practice/virtual-kubelet-practice/pkg/remote"
	"github.com/sirupsen/logrus"
	cli "github.com/virtual-kubelet/node-cli"
	//"github.com/virtual-kubelet/node-cli/opts"
	logruscli "github.com/virtual-kubelet/node-cli/logrus"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
)

const (
	k8sVersion   = "v1.22.0"
	providerName = "example-provider"
)

// 启动命令
// go run main.go --provider example-provider --kubeconfig ./config/config.yaml --nodename mynode

func main() {

	remoteCRI := remote.NewRemoteCRIContainer(common.R, common.I)

	ctx := cli.ContextWithCancelOnSignal(context.Background())
	logger := logrus.StandardLogger()

	log.L = logruslogger.FromLogrus(logrus.NewEntry(logger))
	logConfig := &logruscli.Config{LogLevel: "info"}

	node, err := cli.New(ctx,
		cli.WithProvider(providerName, func(cfg provider.InitConfig) (provider.Provider, error) {
			return providers.NewCriProvider(common.SetupConfig(cfg), remoteCRI), nil
		}),
		cli.WithKubernetesNodeVersion(k8sVersion),
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
