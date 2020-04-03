test:
	bash -c 'diff -u <(echo -n) <(gofmt -s -d .)'
	go vet ./...
	go test ./... -v -covermode=atomic -coverprofile=coverage.out

build:
	GOOS=linux GOARCH=amd64 go build -o build/g8 *.go
	

.PHONY: test
