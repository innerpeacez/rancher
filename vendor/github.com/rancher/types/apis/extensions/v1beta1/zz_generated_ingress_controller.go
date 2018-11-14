package v1beta1

import (
	"context"

	"github.com/rancher/norman/controller"
	"github.com/rancher/norman/objectclient"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	IngressGroupVersionKind = schema.GroupVersionKind{
		Version: Version,
		Group:   GroupName,
		Kind:    "Ingress",
	}
	IngressResource = metav1.APIResource{
		Name:         "ingresses",
		SingularName: "ingress",
		Namespaced:   true,

		Kind: IngressGroupVersionKind.Kind,
	}
)

type IngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []v1beta1.Ingress
}

type IngressHandlerFunc func(key string, obj *v1beta1.Ingress) (runtime.Object, error)

type IngressLister interface {
	List(namespace string, selector labels.Selector) (ret []*v1beta1.Ingress, err error)
	Get(namespace, name string) (*v1beta1.Ingress, error)
}

type IngressController interface {
	Generic() controller.GenericController
	Informer() cache.SharedIndexInformer
	Lister() IngressLister
	AddHandler(ctx context.Context, name string, handler IngressHandlerFunc)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, handler IngressHandlerFunc)
	Enqueue(namespace, name string)
	Sync(ctx context.Context) error
	Start(ctx context.Context, threadiness int) error
}

type IngressInterface interface {
	ObjectClient() *objectclient.ObjectClient
	Create(*v1beta1.Ingress) (*v1beta1.Ingress, error)
	GetNamespaced(namespace, name string, opts metav1.GetOptions) (*v1beta1.Ingress, error)
	Get(name string, opts metav1.GetOptions) (*v1beta1.Ingress, error)
	Update(*v1beta1.Ingress) (*v1beta1.Ingress, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (*IngressList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Controller() IngressController
	AddHandler(ctx context.Context, name string, sync IngressHandlerFunc)
	AddLifecycle(ctx context.Context, name string, lifecycle IngressLifecycle)
	AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync IngressHandlerFunc)
	AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle IngressLifecycle)
}

type ingressLister struct {
	controller *ingressController
}

func (l *ingressLister) List(namespace string, selector labels.Selector) (ret []*v1beta1.Ingress, err error) {
	err = cache.ListAllByNamespace(l.controller.Informer().GetIndexer(), namespace, selector, func(obj interface{}) {
		ret = append(ret, obj.(*v1beta1.Ingress))
	})
	return
}

func (l *ingressLister) Get(namespace, name string) (*v1beta1.Ingress, error) {
	var key string
	if namespace != "" {
		key = namespace + "/" + name
	} else {
		key = name
	}
	obj, exists, err := l.controller.Informer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    IngressGroupVersionKind.Group,
			Resource: "ingress",
		}, key)
	}
	return obj.(*v1beta1.Ingress), nil
}

type ingressController struct {
	controller.GenericController
}

func (c *ingressController) Generic() controller.GenericController {
	return c.GenericController
}

func (c *ingressController) Lister() IngressLister {
	return &ingressLister{
		controller: c,
	}
}

func (c *ingressController) AddHandler(ctx context.Context, name string, handler IngressHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*v1beta1.Ingress); ok {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

func (c *ingressController) AddClusterScopedHandler(ctx context.Context, name, cluster string, handler IngressHandlerFunc) {
	c.GenericController.AddHandler(ctx, name, func(key string, obj interface{}) (interface{}, error) {
		if obj == nil {
			return handler(key, nil)
		} else if v, ok := obj.(*v1beta1.Ingress); ok && controller.ObjectInCluster(cluster, obj) {
			return handler(key, v)
		} else {
			return nil, nil
		}
	})
}

type ingressFactory struct {
}

func (c ingressFactory) Object() runtime.Object {
	return &v1beta1.Ingress{}
}

func (c ingressFactory) List() runtime.Object {
	return &IngressList{}
}

func (s *ingressClient) Controller() IngressController {
	s.client.Lock()
	defer s.client.Unlock()

	c, ok := s.client.ingressControllers[s.ns]
	if ok {
		return c
	}

	genericController := controller.NewGenericController(IngressGroupVersionKind.Kind+"Controller",
		s.objectClient)

	c = &ingressController{
		GenericController: genericController,
	}

	s.client.ingressControllers[s.ns] = c
	s.client.starters = append(s.client.starters, c)

	return c
}

type ingressClient struct {
	client       *Client
	ns           string
	objectClient *objectclient.ObjectClient
	controller   IngressController
}

func (s *ingressClient) ObjectClient() *objectclient.ObjectClient {
	return s.objectClient
}

func (s *ingressClient) Create(o *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	obj, err := s.objectClient.Create(o)
	return obj.(*v1beta1.Ingress), err
}

func (s *ingressClient) Get(name string, opts metav1.GetOptions) (*v1beta1.Ingress, error) {
	obj, err := s.objectClient.Get(name, opts)
	return obj.(*v1beta1.Ingress), err
}

func (s *ingressClient) GetNamespaced(namespace, name string, opts metav1.GetOptions) (*v1beta1.Ingress, error) {
	obj, err := s.objectClient.GetNamespaced(namespace, name, opts)
	return obj.(*v1beta1.Ingress), err
}

func (s *ingressClient) Update(o *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	obj, err := s.objectClient.Update(o.Name, o)
	return obj.(*v1beta1.Ingress), err
}

func (s *ingressClient) Delete(name string, options *metav1.DeleteOptions) error {
	return s.objectClient.Delete(name, options)
}

func (s *ingressClient) DeleteNamespaced(namespace, name string, options *metav1.DeleteOptions) error {
	return s.objectClient.DeleteNamespaced(namespace, name, options)
}

func (s *ingressClient) List(opts metav1.ListOptions) (*IngressList, error) {
	obj, err := s.objectClient.List(opts)
	return obj.(*IngressList), err
}

func (s *ingressClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.objectClient.Watch(opts)
}

// Patch applies the patch and returns the patched deployment.
func (s *ingressClient) Patch(o *v1beta1.Ingress, data []byte, subresources ...string) (*v1beta1.Ingress, error) {
	obj, err := s.objectClient.Patch(o.Name, o, data, subresources...)
	return obj.(*v1beta1.Ingress), err
}

func (s *ingressClient) DeleteCollection(deleteOpts *metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return s.objectClient.DeleteCollection(deleteOpts, listOpts)
}

func (s *ingressClient) AddHandler(ctx context.Context, name string, sync IngressHandlerFunc) {
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *ingressClient) AddLifecycle(ctx context.Context, name string, lifecycle IngressLifecycle) {
	sync := NewIngressLifecycleAdapter(name, false, s, lifecycle)
	s.Controller().AddHandler(ctx, name, sync)
}

func (s *ingressClient) AddClusterScopedHandler(ctx context.Context, name, clusterName string, sync IngressHandlerFunc) {
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}

func (s *ingressClient) AddClusterScopedLifecycle(ctx context.Context, name, clusterName string, lifecycle IngressLifecycle) {
	sync := NewIngressLifecycleAdapter(name+"_"+clusterName, true, s, lifecycle)
	s.Controller().AddClusterScopedHandler(ctx, name, clusterName, sync)
}
