package main

import (
	"ft_cni/internal/load"
	"ft_cni/internal/logs"
	"ft_cni/internal/network"
	"net"
)


var configJSON *load.CNIConfig

func init() {
	logs.Init("/tmp/mycni.log")
	logs.Log.Info("Init Function | Initialized Log")

	logs.Log.Info("env check")
	configJSON = load.LoadFromFile("/etc/cni/net.d/10-ftcni.conflist")
	logs.Log.Info("Logjson check")
	

}

func main() {

	network.EnableNat()
	network.SetupVXLAN("vxlan0", configJSON.IFace, 1001, net.ParseIP("192.168.56.111"), net.ParseIP("192.168.56.110"))

}