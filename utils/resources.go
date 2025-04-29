package utils

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"text/template"
	"time"

	"github.com/hdssbks/mydeployment/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func init() {
	// 初始化全局随机数生成器
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

var funcMap = template.FuncMap{
	"randAlphaNum": RandStr,
	"lower":        strings.ToLower,
}

func NewPod(deployment *v1beta1.MyDeployment) *corev1.Pod {
	pod := &corev1.Pod{}
	tpl, err := template.New("pod.tpl").Funcs(funcMap).ParseFiles("templates/pod.tpl")
	if err != nil {
		log.Fatalln(fmt.Errorf("error parsing template: %v", err))
	}

	buffer := bytes.Buffer{}
	if err = tpl.Execute(&buffer, deployment); err != nil {
		log.Fatalln(fmt.Errorf("error excute template: %v", err))
	}

	err = yaml.Unmarshal(buffer.Bytes(), pod)
	if err != nil {
		log.Fatalln(fmt.Errorf("unable to unmarshal template: %v", err))
	}
	return pod
}
