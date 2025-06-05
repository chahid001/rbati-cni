

all:
	GOOS=linux GOARCH=amd64 go build -o ft_cni main.go
	cp ft_cni ../../Projects/42-Inception-Of-Things/p2/config
init: 
	go mod init ft_cni
