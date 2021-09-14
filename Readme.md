# Shorty - A URL Shortener Web Service
A URL shortener web service written in go and deployable to kubernetes via helm.

## Tech Stack
- Written in [Go](https://golang.org/)
- [Gin Web Framework](https://gin-gonic.com/) 
- [MongoDB database](https://www.mongodb.com) via [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) 
- [Testify Testing Framework](https://github.com/stretchr/testify)
- [Docker](https://www.docker.com/), [Kubernetes](https://kubernetes.io/), [Helm](https://helm.sh/)
- [OpenAPI](https://www.openapis.org/) API specification with [Swagger-UI](https://swagger.io/) renderer. 

## Installation

Installing the service requires that the `docker`, `kubectl` and `helm` tools are locally configured.  
In that case you can build and install the service on your k8s cluster using the following commands.
Alternatively you can run it locally as described under [Running the service locally](#running-the-service-locally).

```bash
# Build the docker image tagged shorty:latest
docker build -t shorty:latest .

# If k8s is not running on you local docker instance push the image to the appropriate repository.
# If required change the image name/repository above and in ./helm/shorty/variables.yaml
# docker push shorty:latest

# Download helm dependencies (mongodb)
helm dep update ./helm/shorty

# Install the helm chart
helm install shorty ./helm/shorty

# After any changes upgrade the release via
docker build -t shorty:latest .
helm upgrade shorty ./helm/shorty
```


## Development
### Using Helm
Install as above and after any changes upgrade the release via
```bash
docker build -t shorty:latest .
helm upgrade shorty ./helm/shorty
```

### Running the service locally
You can also run and test the service locally if you have go installed and a locally accessible MongoDB instance.
- Download the go dependencies via `go mod download`
- Get swagger-ui-dist locally by running
  ```bash 
  wget https://github.com/swagger-api/swagger-ui/archive/refs/tags/v3.52.1.tar.gz
  tar -xzf v3.52.1.tar.gz
  mv swagger-ui-3.52.1/dist swagger-dist
  rm -rf swagger-ui-3.52.1 v3.52.1.tar.gz
  sed -i 's+https://petstore.swagger.io/v2/swagger.json+shorty.yaml+g' swagger-dist/index.html
  ln -s ../api/shorty.yaml swagger-dist/shorty.yaml
  ```

- Make sure `MONGO_URL` points to an accessible MongoDB instance where the user has read/write access to the database `shorty` or set `SHORTY_DB` accordingly. Tests use the database `testing` instead, change in `main_test.go` if necessary. 
  - You can run a MongoDB locally in docker via:
    ```
    docker pull mongo
    docker run --name mongodb \
      -e MONGO_INITDB_ROOT_USERNAME=shorty \
      -e MONGO_INITDB_ROOT_PASSWORD=short \
      -p 27017:27017 \
      -d mongo:latest
    export MONGO_URL=mongodb://shorty:short@localhost:27017
    ```
- Test the service via `go test .`
- Start the service via `go run .`
- Visit http://localhost:8080/api to explore the API
- You can change the port by setting the `PORT` environment variable.
## Documentation
- See [api/shorty.yaml](api/shorty.yaml) for the OpenAPI specification of the service. Also served rendered under `/api`.
- See [helm/Notes.md](helm/Notes.md) for notes about how the Helm chart was created.

## Author and License

Copyright (c) 2021 Benedikt Nordhoff

Released under the MIT license. See [LICENSE](LICENSE) for details.
