package network

import (
	"fmt"
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


func EnableNat(localPodCIDR, nodeIface string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to get iptables instance: %w", err)
	}

	if err := ipt.ChangePolicy("filter", "FORWARD", "ACCEPT"); err != nil {
		log.Printf("[WARN] Could not set FORWARD policy to ACCEPT: %v. This might be okay.", err)
	}

	err = ipt.AppendUnique("nat", "POSTROUTING",
		"-s", localPodCIDR,
		"-o", nodeIface,
		"-j", "MASQUERADE",
	)
	if err != nil {
		return fmt.Errorf("failed to add masquerade rule: %w", err)
	}

	log.Println("[INFO] Successfully enabled NAT for external traffic.")
	return nil
}