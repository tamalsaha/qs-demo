package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/gorilla/schema"
	"net/url"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func main() {
	err := indexPods()
	if err != nil {
		panic(err)
	}
}

func NewClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	ctrl.SetLogger(klogr.New())
	cfg := ctrl.GetConfigOrDie()
	cfg.QPS = 100
	cfg.Burst = 100

	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
		//Opts: client.WarningHandlerOptions{
		//	SuppressWarnings:   false,
		//	AllowDuplicateLogs: false,
		//},
	})
}

func indexPods() error {
	fmt.Println("Using kubebuilder client")
	kc, err := NewClient()
	if err != nil {
		return err
	}

	var pods unstructured.UnstructuredList
	pods.SetAPIVersion("v1")
	pods.SetKind("Pod")
	err = kc.List(context.TODO(), &pods)
	if err != nil {
		return err
	}

	var encoder = schema.NewEncoder()
	encoder.SetAliasTag("json")

	for _, obj := range pods.Items {
		obj.SetManagedFields(nil)

		form := url.Values{}
		err := encoder.Encode(obj, form)
		if err != nil {
			panic(err)
		}
		fmt.Println(form.Encode())

		break
	}
	return nil
}

func PrimaryKey(clusterUID string, obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	oid := fmt.Sprintf("C=%s,G=%s,K=%s,NS=%s,N=%s", clusterUID, gvk.Group, gvk.Kind, obj.GetNamespace(), obj.GetName())
	return fmt.Sprintf("%x", md5.Sum([]byte(oid)))
}
