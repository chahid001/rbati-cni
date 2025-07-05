package network

import (
	"log"
	"net"

	"github.com/coreos/go-iptables/iptables"
)

func GetGatewayIP(podSubnet string) (string, error) {
	
	_, ipNet, err := net.ParseCIDR(podSubnet)
	if err != nil {
		return "", err
	}

	gatewayIP := make(net.IP, len(ipNet.IP))
	copy(gatewayIP, ipNet.IP)

	gatewayIP[len(gatewayIP)-1]++

	return gatewayIP.String(), nil
}

func EnableNat(localPodCIDR, clusterCIDR, bridgeName string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	err = ipt.AppendUnique(
		"mangle", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--set-mss", "1410",
	)
	if err != nil {
		log.Printf("[ERROR] NAT Error: Failed to add TCPMSS clamp rule: %v", err)
	}

	err = ipt.Insert("filter", "FORWARD", 1, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	if err != nil {
		log.Printf("[ERROR] NAT Error: Failed to add conntrack rule: %v", err)
	}

	err = ipt.Insert("filter", "FORWARD", 2, "-i", bridgeName, "-j", "ACCEPT")
	if err != nil {
		log.Printf("[ERROR] NAT Error: %v", err)
	}

	err = ipt.Insert("filter", "FORWARD", 3, "-o", bridgeName, "-j", "ACCEPT")
	if err != nil {
		log.Printf("[ERROR] NAT Error: %v", err)
	}

	err = ipt.AppendUnique(
		"nat", "POSTROUTING",
		"-s", localPodCIDR,
		"!", "-d", clusterCIDR,
		"-j", "MASQUERADE",
	)
	if err != nil {
		return err
	}

	log.Println("[INFO] Successfully enabled NAT and forwarding rules.")
	return nil
}
