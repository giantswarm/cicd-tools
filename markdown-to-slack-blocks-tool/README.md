This image converts markdown into slack block code .


```
IMAGE=quay.io/giantswarm/tinkerers-ci:m2b
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
