package utils

import (
	"bytes"
	"fmt"
	"github.com/hdssbks/mydeployment/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"log"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"randAlphaNum": func(n int) string {
		const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
		result := make([]byte, n)
		for i := range result {
			result[i] = letters[i%len(letters)]
		}
		return string(result)
	},
	"lower": strings.ToLower,
}

func NewPod(deployment *v1beta1.MyDeployment) *corev1.Pod {
	pod := &corev1.Pod{}
	tpl, err := template.New("pod").Funcs(funcMap).ParseFiles("templates/pod.tpl")
	if err != nil {
		log.Fatalln(fmt.Errorf("error parsing template: %v", err))
	}

	buffer := bytes.Buffer{}
	if err := tpl.Execute(&buffer, deployment); err != nil {
		log.Fatalln(fmt.Errorf("error excute template: %v", err))
	}

	err = yaml.Unmarshal(buffer.Bytes(), pod)
	if err != nil {
		panic("unable to unmarshal template")
	}
	return pod
}
