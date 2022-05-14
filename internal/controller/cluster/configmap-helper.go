package cluster

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapName      = "cdevents-provider-status-list"
	configMapNameSpace = "crossplane-system"
)

func registerClusterCreationSuccessEvent(ctx context.Context, c client.Client, key string) error {
	var cm corev1.ConfigMap

	objectKey := client.ObjectKey{
		Name:      configMapName,
		Namespace: configMapNameSpace,
	}

	err := c.Get(ctx, objectKey, &cm)

	if err != nil {
		return err
	}

	// handle the first key case
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	cm.Data[key] = "updated"
	err = c.Update(ctx, &cm)
	if err != nil {
		return err
	}

	return nil
}

func checkClusterCreationSuccessEvent(ctx context.Context, c client.Client, key string) (bool, error) {
	var cm corev1.ConfigMap

	objectKey := client.ObjectKey{
		Name:      configMapName,
		Namespace: configMapNameSpace,
	}

	err := c.Get(ctx, objectKey, &cm)
	if err != nil {
		return false, err
	}

	if _, ok := cm.Data[key]; ok {
		return true, nil
	}

	return false, nil
}
