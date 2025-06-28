

all:
	protoc --go_out=api --go-grpc_out=api api/ipam.proto
	
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-cni cmd/cni/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-agent cmd/agent/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-controller cmd/controller/main.go

	cp bin/rbati-cni ../../Projects/42-Inception-Of-Things/p2/config
	cp bin/rbati-agent ../../Projects/42-Inception-Of-Things/p2/config
	cp bin/rbati-controller ../../Projects/42-Inception-Of-Things/p2/config
init: 
	go mod init rbati-cni
