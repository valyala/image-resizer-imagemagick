build:
	go build ./image-resizer.go

clean:
	rm -f ./image-resizer

update-libs:
	go get -u github.com/mitchellh/goamz/aws
	go get -u github.com/mitchellh/goamz/s3
	go get -u github.com/valyala/ybc/bindings/go/ybc
