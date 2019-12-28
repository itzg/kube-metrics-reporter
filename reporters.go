package main

import (
	"context"
	"fmt"
	lpsender "github.com/itzg/line-protocol-sender"
	"go.uber.org/zap"
	"io"
	"time"
)

type Reporter interface {
	Start(namespace string) Batch
}

type Batch interface {
	io.Closer
	// cpuUsage is millicores and memUsage is megabytes
	Report(podName, containerName string, cpuUsage, memUsage int64)
}

type StdoutReporter struct{}

type StdoutBatch struct {
	timestamp time.Time
	namespace string
}

func (r StdoutReporter) Start(namespace string) Batch {
	return &StdoutBatch{
		timestamp: time.Now(),
		namespace: namespace,
	}
}

func (s *StdoutBatch) Close() error {
	return nil
}

func (s *StdoutBatch) Report(podName, containerName string, cpuUsage, memUsage int64) {
	fmt.Printf("%s pod=%s, container=%s, cpu=%dm, mem=%dMi\n",
		s.timestamp.Format(time.RFC3339), podName, containerName, cpuUsage, memUsage)
}

type TelegrafReporter struct {
	client lpsender.Client
}

type telegrafBatch struct {
	client    lpsender.Client
	timestamp time.Time
	namespace string
}

func (t *telegrafBatch) Close() error {
	t.client.Flush()
	return nil
}

func (t *TelegrafReporter) Start(namespace string) Batch {
	return &telegrafBatch{
		client:    t.client,
		timestamp: time.Now(),
		namespace: namespace,
	}
}

func (t *telegrafBatch) Report(podName, containerName string, cpuUsage, memUsage int64) {
	m := lpsender.NewSimpleMetric("kubernetes_pod_container")
	m.SetTime(t.timestamp)

	m.AddTag("namespace", t.namespace)
	m.AddTag("pod_name", podName)
	m.AddTag("container_name", containerName)

	m.AddField("cpu_usage_millicores", cpuUsage)
	m.AddField("memory_usage_mbytes", memUsage)

	t.client.Send(m)
}

func NewTelegrafReporter(telegrafEndpoint string, logger *zap.SugaredLogger) (*TelegrafReporter, error) {
	client, err := lpsender.NewClient(context.Background(), lpsender.Config{
		Endpoint:     telegrafEndpoint,
		BatchTimeout: 10 * time.Second,
		ErrorListener: func(err error) {
			logger.Errorw("failed to send metrics", "err", err)
		},
	})
	if err != nil {
		return nil, err
	}
	return &TelegrafReporter{client: client}, nil
}
