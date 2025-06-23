

all:
	GOOS=linux GOARCH=amd64 go build -o bin/rbati-cni cmd/cni/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/rbati-agent cmd/controller/main.go
	cp bin/rbati-cni ../../Projects/42-Inception-Of-Things/p2/config
	cp bin/rbati-agent ../../Projects/42-Inception-Of-Things/p2/config
init: 
	go mod init ft_cni
