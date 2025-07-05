

all:
	
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-d cmd/cni/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-agent cmd/agent/main.go

	scp -i ~/.ssh/id_rsa bin/rbati-* schahid@192.168.0.170:/home/schahid/workspace/p2/config

init: 
	go mod init rbati-cni
