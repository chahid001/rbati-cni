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

func EnableNat(localPodCIDR, clusterCIDR string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to get iptables instance: %w", err)
	}

	if err := ipt.ChangePolicy("filter", "FORWARD", "ACCEPT"); err != nil {
		return fmt.Errorf("failed to set FORWARD policy to ACCEPT: %w", err)
	}

	ruleEstablished := []string{"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"}
	if err := ipt.Insert("filter", "FORWARD", 1, ruleEstablished...); err != nil {
		return fmt.Errorf("failed to insert established/related rule: %w", err)
	}
	
	if err := ipt.Insert("filter", "FORWARD", 2, "-j", "ACCEPT"); err != nil {
		return fmt.Errorf("failed to insert blanket ACCEPT rule: %w", err)
	}

	err = ipt.AppendUnique("nat", "POSTROUTING",
		"-s", localPodCIDR,
		"!", "-d", clusterCIDR,
		"-j", "MASQUERADE")
	if err != nil {
		return fmt.Errorf("failed to add masquerade rule: %w", err)
	}

	log.Println("[INFO] Successfully enabled simplified NAT and forwarding rules.")
	return nil
}