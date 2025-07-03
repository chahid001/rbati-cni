

all:
	protoc --go_out=api --go-grpc_out=api api/ipam.proto
	
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-cni cmd/cni/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-agent cmd/agent/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-controller cmd/controller/main.go

	scp -i ~/.ssh/id_rsa bin/rbati-* schahid@192.168.0.170:/home/schahid/workspace/p2/config

init: 
	go mod init rbati-cni
