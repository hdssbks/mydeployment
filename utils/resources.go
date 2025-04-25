package utils

import (
	"bytes"
	"github.com/hdssbks/mydeployment/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"text/template"
)

func NewPod(deployment *v1beta1.MyDeployment) *corev1.Pod {
	pod := &corev1.Pod{}
	tpl, err := template.ParseFiles("templates/pod.yaml")
	if err != nil {
		panic("unable to parse template")
	}

	buffer := bytes.Buffer{}
	if err := tpl.Execute(&buffer, deployment); err != nil {
		panic("unable to execute template")
	}

	err = yaml.Unmarshal(buffer.Bytes(), pod)
	if err != nil {
		panic("unable to unmarshal template")
	}
	return pod
}
