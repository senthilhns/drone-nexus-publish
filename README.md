# drone-nexus-publish

Drone plugin to publish artifacts to Nexus Repository Manager.

## Docker

Build the Docker image with the following commands:

```
docker buildx build -t DOCKER_ORG/drone-nexus-publish --platform linux/amd64 .
```

Please note incorrectly building the image for the correct x64 linux and with
CGO disabled will result in an error when running the Docker image:

```
docker: Error response from daemon: Container command
'/bin/drone-nexus-publish' not found or does not exist..
```

## Usage

```bash
docker run --rm \
  -e PLUGIN_USERNAME=${username} \
  -e PLUGIN_PASSWORD=${password} \
  -e PLUGIN_SERVER_URL=http://nexus-publish.server \
  -e PLUGIN_FILENAME=./target/example.jar \
  -e PLUGIN_FORMAT=maven2 \
  -e PLUGIN_REPOSITORY=maven-releases \
  -e PLUGIN_ATTRIBUTES="-CgroupId=org.testing -CartifactId=example -Cversion=1.0 -Aextension=jar -Aclassifier=bin" \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  harnesscommunity/drone-nexus-publish
```



In Harness CI,
```yaml
              - step:
                  type: Plugin
                  name: Plugin_1
                  identifier: Plugin_1
                  spec:
                    connectorRef: harnessnew
                    image: harnesscommunity/publish-nexus-repository:1.1.1
                    settings:
                      username: deploy-user
                      password: testing-nexus
                      server_url: http://nexus-publish.server
                      filename: ./target/example.jar
                      format: maven2
                      repository: maven-releases
                      attributes: "-CgroupId=org.testing -CartifactId=example -Cversion=1.0 -Aextension=jar -Aclassifier=bin"
```
