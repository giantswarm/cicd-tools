This image converts markdown into slack block code .


```
IMAGE=quay.io/giantswarm/markdown-to-slack-blocks-tool:latest
```

# Build container

```
docker build -t ${IMAGE} .
```

# Publish container

```
docker push ${IMAGE} 
```

# Using the container

```
docker run -e INPUT="#test"  -it  ${IMAGE} 
```
