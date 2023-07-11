package main

import (
	"context"
	"github.com/practice/virtual-kubelet-practice/pkg/common"
	"github.com/practice/virtual-kubelet-practice/pkg/providers"
	"github.com/sirupsen/logrus"
	cli "github.com/virtual-kubelet/node-cli"
	//"github.com/virtual-kubelet/node-cli/opts"
	logruscli "github.com/virtual-kubelet/node-cli/logrus"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// FIXME: 配置bug

	remoteCRI := providers.NewRemoteCRIContainer(common.R, common.I)

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
			return providers.NewCriProvider(common.SetupConfig(cfg), remoteCRI), nil
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
