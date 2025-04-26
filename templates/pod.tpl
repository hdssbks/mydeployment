apiVersion: v1
kind: Pod
metadata:
  name: {{.ObjectMeta.Name}}-{{randAlphaNum 5 | lower}}
  namespace: {{.ObjectMeta.Namespace}}
  labels:
    {{- range $key, $value := .Spec.Selector.MatchLabels}}
    {{$key}}: {{$value}}
    {{- end}}
spec:
  containers:
    {{- range .Spec.Template.Spec.Containers}}
    - name: {{.Name}}-{{randAlphaNum 5 | lower}}
      image: {{.Image}}
      ports:
        {{- range .Ports}}
        - containerPort: {{.ContainerPort}}
        {{- end}}
    {{- end}}