FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/site-router"]

COPY . /go/src/github.com/mikroio/site-router
RUN apk-install -t build-deps go git mercurial \
	&& cd /go/src/github.com/mikroio/site-router \
	&& export GOPATH=/go \
	&& go get \
	&& go build -ldflags "-X main.Version $(cat VERSION)" -o /bin/site-router \
	&& rm -rf /go \
	&& apk del --purge build-deps
