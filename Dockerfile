FROM golang:1.7.1-alpine

# install curl 
RUN apk add --update curl && rm -rf /var/cache/apk/*

# copy deps
ADD vendor /go/src/

# copy sources
RUN mkdir /goc-proxy 
ADD . /goc-proxy/ 

# build
WORKDIR /goc-proxy/gocp/

#BUILD_DATE=$(date -u +%Y%m%d.%H%M%S)
#GIT_COMMIT=$(git rev-parse HEAD)
#GIT_BRANCH=$(git symbolic-ref --short HEAD)
#BUILD_FLAGS=-X main.BuildDate=$(BUILD_DATE) -X main.Revision=$(GIT_COMMIT) -X main.Branch=$(GIT_BRANCH)
#go build -ldflags "$(BUILD_FLAGS)" -o gocp .

RUN go build -o gocp .


HEALTHCHECK CMD curl --fail http://localhost:8000/_/status || exit 1

EXPOSE 8000/tcp

env PATH /goc-proxy/gocp:$PATH

# run
CMD ["gocp"]
