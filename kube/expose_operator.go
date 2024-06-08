package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	hijackAnnotation      = "meta.ksrp-expose/hijack"
	defaultSpecAnnotation = "meta.ksrp-expose/default-spec"
	managedByLabel        = "app.kubernetes.io/managed-by"
)

var (
	serviceGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "Service",
	}
)

type ExposeOperator struct {
	name      string
	kc        *Client
	namespace string
}

func (o *ExposeOperator) newService(serviceName string, appName string, port int) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"annotations": map[string]string{
					hijackAnnotation: "true",
				},
				"labels": map[string]string{
					"app":          serviceName,
					managedByLabel: o.name,
				},
				"name":      serviceName,
				"namespace": o.namespace,
			},
			"spec": map[string]any{
				"selector": map[string]any{
					"app": appName,
				},
				"type": "ClusterIP",
				"ports": []any{
					map[string]any{
						"port":       port,
						"protocol":   "TCP",
						"targetPort": port,
					},
				},
			},
		},
	}
}

func (o *ExposeOperator) HijackService(ctx context.Context, serviceName string, appName string, port int) error {
	obj, err := o.kc.Get(ctx, serviceGVK, o.namespace, serviceName)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if obj == nil {
		_, err := o.kc.Create(ctx, o.newService(serviceName, appName, port))
		return err
	}

	if obj.GetLabels()[managedByLabel] != o.name {
		return fmt.Errorf("service %s not managed", serviceName)
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[hijackAnnotation] = "true"
	obj.SetAnnotations(annotations)

	unstructured.SetNestedStringMap(obj.Object, map[string]string{"app": appName}, "spec", "selector")
	unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"port": int64(port), "protocol": "TCP", "targetPort": int64(port)},
	}, "spec", "ports")

	_, err = o.kc.Update(ctx, obj)
	return err
}

func (o *ExposeOperator) RestoreService(ctx context.Context, serviceName string) error {
	obj, err := o.kc.Get(ctx, serviceGVK, o.namespace, serviceName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if obj.GetLabels()[managedByLabel] != o.name {
		return fmt.Errorf("service %s not managed", serviceName)
	}
	annotations := obj.GetAnnotations()
	if annotations[hijackAnnotation] != "true" {
		return nil
	}

	// 删除服务或还原默认 selector 和 ports
	specData, ok := annotations[defaultSpecAnnotation]
	if !ok {
		return o.kc.Delete(ctx, obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName())
	}

	dec := json.NewDecoder(strings.NewReader(specData))
	dec.UseNumber()
	var defaultSpec map[string]any
	err = dec.Decode(&defaultSpec)
	if err != nil {
		return err
	}

	delete(annotations, hijackAnnotation)
	obj.SetAnnotations(annotations)

	for key, value := range defaultSpec {
		unstructured.SetNestedField(obj.Object, value, "spec", key)
	}

	_, err = o.kc.Update(ctx, obj)
	return err
}

func NewExposeOperator(name string, kc *Client, namespace string) *ExposeOperator {
	return &ExposeOperator{
		name:      name,
		kc:        kc,
		namespace: namespace,
	}
}
