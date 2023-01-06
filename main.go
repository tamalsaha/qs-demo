package main

import (
	"context"
	"crypto/md5"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cu "kmodules.xyz/client-go/client"

	"github.com/meilisearch/meilisearch-go"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func main() {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host: "http://localhost:7700",
	})

	// https://docs.meilisearch.com/learn/core_concepts/primary_key.html#primary-field
	// md5("C=%s," +"G=%s,K=%s,NS=%s,N=%s")

	client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        "k8s",
		PrimaryKey: "oid",
	})

	//documents := []map[string]interface{}{
	//	{
	//		"reference_number": 287947,
	//		"title":            "Diary of a Wimpy Kid",
	//		"author":           "Jeff Kinney",
	//		"genres":           []string{"comedy", "humor"},
	//		"price":            5.00,
	//	},
	//}
	//client.Index("k8s").AddDocuments(documents, "reference_number")

	err := indexPods(client.Index("k8s"))
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

func indexPods(index *meilisearch.Index) error {
	fmt.Println("Using kubebuilder client")
	kc, err := NewClient()
	if err != nil {
		return err
	}

	clusterUID, err := cu.ClusterUID(kc)
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

	documents := make([]map[string]interface{}, 0, len(pods.Items))
	for _, obj := range pods.Items {
		obj.SetManagedFields(nil)
		doc := obj.UnstructuredContent()
		doc["oid"] = PrimaryKey(clusterUID, &obj)

		documents = append(documents, doc)
	}
	index.AddDocuments(documents, "oid")
	return nil
}

func PrimaryKey(clusterUID string, obj client.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	oid := fmt.Sprintf("C=%s,G=%s,K=%s,NS=%s,N=%s", clusterUID, gvk.Group, gvk.Kind, obj.GetNamespace(), obj.GetName())
	return fmt.Sprintf("%x", md5.Sum([]byte(oid)))
}
