package main

import (
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sync"
)

type LabelResolver interface {
	Resolve(podName string) map[string]string
}

type WatchingLabelResolver struct {
	podInterface clientv1.PodInterface
	logger       *zap.SugaredLogger

	// labels maps pod name to labels
	labels     map[string]map[string]string
	labelsLock sync.RWMutex
}

func NewWatchingLabelResolver(c *rest.Config, namespace string, logger *zap.SugaredLogger) (*WatchingLabelResolver, error) {
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	podInterface := clientset.CoreV1().Pods(namespace)

	w := &WatchingLabelResolver{
		podInterface: podInterface,
		logger:       logger.Named("label_resolver"),
		labels:       make(map[string]map[string]string),
	}
	go w.watch()
	return w, nil
}

func (w *WatchingLabelResolver) Resolve(podName string) map[string]string {
	w.labelsLock.RLock()
	defer w.labelsLock.RUnlock()
	return w.labels[podName]
}

func (w *WatchingLabelResolver) watch() {
	watchIf, err := w.podInterface.Watch(metav1.ListOptions{})
	if err != nil {
		w.logger.Errorw("failed to watch pods", "err", err)
		return
	}
	defer watchIf.Stop()

	for {
		e := <-watchIf.ResultChan()
		pod := e.Object.(*corev1.Pod)
		switch e.Type {
		case watch.Added, watch.Modified:
			w.labelsLock.Lock()
			w.labels[pod.Name] = pod.Labels
			w.labelsLock.Unlock()
		case watch.Deleted:
			w.labelsLock.Lock()
			delete(w.labels, pod.Name)
			w.labelsLock.Unlock()
		}
	}
}

type DisabledLabelResolver struct{}

func (d *DisabledLabelResolver) Resolve(string) map[string]string {
	return nil
}
