package main

import (
	"fmt"
	"github.com/itzg/go-flagsfiller"
	"github.com/itzg/zapconfigs"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"time"
)

var config struct {
	Namespace string `default:"default" usage:"the namespace of the pods to collect"`
	Interval time.Duration `default:"1m" usage:"the interval of metrics collection"`
	Repeat bool `usage:"indicates console reporting should repeat at the given interval"`
	Telegraf struct {
		Endpoint string `usage:"if configured, metrics will be sent as line protocol to telegraf"`
	}
}

func main() {

	logger := zapconfigs.NewDefaultLogger().Sugar()
	defer logger.Sync()

	err := flagsfiller.Parse(&config, flagsfiller.WithEnv(""))
	if err != nil {
		logger.Fatalw("parsing flags", "err", err)
	}

	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		logger.Fatalw("loading kubeConfig", "err", err)
	}

	clientset, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("creating kube clientset", "err", err)
	}

	podMetricsAccessor := clientset.MetricsV1beta1().PodMetricses(config.Namespace)

	var reporters []Reporter

	if config.Telegraf.Endpoint != "" {
		reporter, err := NewTelegrafReporter(config.Telegraf.Endpoint, logger)
		if err != nil {
			logger.Fatalw("creating telegraf reporter", "err", err)
		}
		reporters = append(reporters, reporter)
		config.Repeat = true

		logger.Infow("reporting metrics to telegraf",
			"endpoint", config.Telegraf.Endpoint,
			"interval", config.Interval)
	}

	if len(reporters) == 0 {
		reporters = append(reporters, &StdoutReporter{})
	}

	if config.Repeat {
		for {
			err = collect(podMetricsAccessor, reporters, config.Namespace)
			if err != nil {
				logger.Error("err", err)
			}
			time.Sleep(config.Interval)
		}
	} else {
		err = collect(podMetricsAccessor, reporters, config.Namespace)
		if err != nil {
			logger.Error("err", err)
		}
	}
}

func collect(podMetricsAccessor v1beta1.PodMetricsInterface, reporters []Reporter, namespace string) error {
	podMetricsList, err := podMetricsAccessor.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list kube metrics: %w", err)
	}

	batches := make([]Batch, len(reporters))
	for i, reporter := range reporters {
		batches[i] = reporter.Start(namespace)
	}

	for _, p := range podMetricsList.Items {
		podName := p.Name
		for _, c := range p.Containers {
			containerName := c.Name
			// matching the units reported by kubectl top pods
			cpuUsage := c.Usage.Cpu().ScaledValue(resource.Milli)
			memUsage := c.Usage.Memory().ScaledValue(resource.Mega)
			for _, batch := range batches {
				batch.Report(podName, containerName, cpuUsage, memUsage)
			}
		}
	}

	for _, batch := range batches {
		_ = batch.Close()
	}

	return nil
}
