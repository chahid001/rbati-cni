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


func EnableNat(localPodCIDR, vxlanIface string) error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to get iptables instance: %w", err)
	}

	customChain := "RBATI-NAT"

	if err := ipt.NewChain("nat", customChain); err != nil {
		if e, ok := err.(*iptables.Error); !ok || e.ExitStatus() != 1 {
			return fmt.Errorf("failed to create custom chain %s: %w", customChain, err)
		}
	}

	jumpRule := []string{"-s", localPodCIDR, "-j", customChain}
	if err := ipt.AppendUnique("nat", "POSTROUTING", jumpRule...); err != nil {
		return fmt.Errorf("failed to add jump rule to custom chain: %w", err)
	}

	if err := ipt.ClearChain("nat", customChain); err != nil {
		return fmt.Errorf("failed to flush custom chain %s: %w", customChain, err)
	}

	natRule := []string{"!", "-o", vxlanIface, "-j", "MASQUERADE"}
	if err := ipt.Append("nat", customChain, natRule...); err != nil {
		return fmt.Errorf("failed to add masquerade rule to custom chain: %w", err)
	}

	log.Println("[INFO] Successfully reconciled NAT for external traffic.")
	return nil
}