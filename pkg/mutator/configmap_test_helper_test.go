package mutator

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newFakeConfigMap(name, namespace, configKey, configData string, annotations map[string]string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Data: map[string]string{
			configKey: configData,
		},
	}
	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}
	return cm
}
