# Notes about the Helm setup

## Setting up the Helm Chart
- Create a fresh chart:
  ```
  helm create shorty
  ```

- Configure the chart to use the `shorty:latest` image in `values.yaml`
  ```yaml
  image:
    repository: shorty  
    tag: "latest"
  ```

- Add MongoDB as a dependency to `Chart.yaml`
  ```yaml
  dependencies:
  - name: mongodb
    version: 10.25.2
    repository: https://charts.bitnami.com/bitnami
  ```

- Download the MongoDB chart
  ```
  helm dep update ./shorty
  ```

### Settings for the shorty service
- Add shorty configuration to `values.yaml`
  ```yaml
  shorty:
    # Set to true to enable Gin release mode
    production: false
    mongo: 
      databaseName: shorty
      collectionName: shorts
  ```

- Setup environment variables
  
  Add `SHORTY_DB`, `SHORTY_COLLECTION` and `GIN_MODE` environment variable in `deployment.yaml/spec/template/spec/containers/-name/env`.

  ```yaml
  env: 
  - name: SHORTY_DB
    value: "{{ .Values.shorty.mongo.databaseName}}"
  - name: SHORTY_COLLECTION
    value: "{{ .Values.shorty.mongo.collectionName}}"
  {{- if .Values.shorty.production }}
  - name: GIN_MODE
    value: "release"
  {{- end }}
  ```
- In the same section adapt the port and the liveness/readiness probes:
  ```yaml
  ports:
    - name: http
      containerPort: 8080
      protocol: TCP
  livenessProbe:
    httpGet:
      path: /check/something
      port: http
  readinessProbe:
    httpGet:
      path: /check/something
      port: http
  ```
- Timestamp the deployment in `deployment.yaml/spec/template/metadata` so that kubernetes redeploys the shorty pods after `helm upgrade`, even if the image tag did not change.
  ```yaml
  annotations:
    timestamp: {{ now | date "20060102150405" | quote}}
    {{- with .Values.podAnnotations }}
    {{- toYaml . | nindent 8 }}
    {{- end }}
  ```


### Settings for MongoDB

- Add MongoDB configuration to `values.yaml`
  ```yaml
  mongodb:
    auth:
      enabled: true
      username: shorty
      password: short
      rootPassword: nooneknows
      database: admin
    service:
      port: 27017
  ```

- Add the `shorty.mongodb.fullname` template to `_helpers.tpl`. 
  ```yaml
  {{- define "shorty.mongodb.fullname" -}}
  {{- printf "%s-%s" .Release.Name "mongodb" | trunc 63 | trimSuffix "-" -}}
  {{- end -}}
  ```

- Create a `secret.yaml` in `templates` storing the `mongo-uri`
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: {{ template "shorty.fullname" . }}
    labels:
      {{- include "shorty.labels" . | nindent 4 }}  
  type: Opaque
  data:
    {{- $fullname := include "shorty.mongodb.fullname" . -}}
    {{- $username := "root" }}
    {{- $password := required "Please specify a MongoDB root password" .Values.mongodb.auth.rootPassword }}
    {{- $dbname := required "Please specify a MongoDB auth database" .Values.mongodb.auth.database }}
    {{- $port := .Values.mongodb.service.port }}              
    mongo-uri: {{ printf "mongodb://%s:%s@%s:%.0f/%s" $username $password $fullname $port $dbname | b64enc | quote }}
  ```
  **TODO:** Setup a proper MongoDB user with read/write access to `shorty.mongo.databaseName` instead of using `root`.  

- Add the `MONGO_URL` environment variable in `deployment.yaml/spec/template/spec/containers/-name/env`.
  ```yaml
  env: 
    - name: MONGO_URL
      valueFrom:
        secretKeyRef:
          name: {{ template "shorty.fullname" . }}
            key: mongo-uri
  ```


## Install the helm chart
```
helm install shorty ./shorty
```

Visit http://localhost:8080/api explore the api and test the service.