package catalog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/registry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const ServiceName = "Catalog"

type Service struct {
	Client  *kubernetes.Clientset
	catalog dtos.Catalog
}

func (s *Service) Init() error {
	clientset, err := connect()
	if err != nil {
		return err
	}

	s.Client = clientset
	return nil
}

func init() {
	registry.Register(&registry.Descriptor{
		Name:         ServiceName,
		Instance:     &Service{},
		InitPriority: registry.High,
	})
}

func (s *Service) Run(ctx context.Context) error {
	svc, err := s.getServiceForDeployment(ctx, "", "")
	if err != nil {
	}
	_, err = s.getPodsForSvc(ctx, svc, "")
	if err != nil {
	}
	return s.startServiceInformer()
}

func connect() (*kubernetes.Clientset, error) {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		return nil, errors.New("could not get filepath of kube config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func (s *Service) startServiceInformer() error {
	factory := informers.NewSharedInformerFactory(s.Client, time.Second)
	stopper := make(chan struct{})
	defer close(stopper)

	// https://pkg.go.dev/k8s.io/client-go@v0.21.2/informers/core/v1#NewServiceInformer
	inf := factory.Core().V1().Services().Informer()
	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*corev1.Service)
			log.Printf("service deleted: %s", mObj.GetName())
		},
		AddFunc: func(obj interface{}) {
			// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
			// interface that allows us to get metadata easily
			mObj := obj.(*corev1.Service)
			log.Printf("New Service Added to Store: %s", mObj.GetName())
		},
	})

	inf.Run(stopper)

	return nil
}

func (s *Service) getServiceForDeployment(ctx context.Context, deployment string, namespace string) (*corev1.Service, error) {
	listOptions := metav1.ListOptions{}
	svcs, err := s.Client.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		log.Fatal(err)
	}
	for _, svc := range svcs.Items {
		if strings.Contains(svc.Name, deployment) {
			fmt.Fprintf(os.Stdout, "service name: %v\n", svc.Name)
			return &svc, nil
		}
	}
	return nil, errors.New("cannot find service for deployment")
}

func (s *Service) getPodsForSvc(ctx context.Context, svc *corev1.Service, namespace string) (*corev1.PodList, error) {
	set := labels.Set(svc.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := s.Client.CoreV1().Pods(namespace).List(ctx, listOptions)
	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)
	}
	return pods, err
}
