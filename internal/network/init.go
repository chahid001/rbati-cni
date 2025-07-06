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

func EnableNat(localPodCIDR, clusterCIDR, bridgeName, vxlanName string) error {
    ipt, err := iptables.New()
    if err != nil {
        return fmt.Errorf("failed to get iptables instance: %w", err)
    }

    cniChain := "RBATI-FORWARD"
    if err := ipt.NewChain("filter", cniChain); err != nil {
        if e, ok := err.(*iptables.Error); !ok || e.ExitStatus() != 1 {
            return fmt.Errorf("failed to create custom chain %s: %w", cniChain, err)
        }
    }

    jumpRule := []string{"-j", cniChain}
    if err := ipt.AppendUnique("filter", "FORWARD", jumpRule...); err != nil {
        return fmt.Errorf("failed to add jump rule to custom chain: %w", err)
    }


    ruleEstablished := []string{"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"}
    if err := ipt.AppendUnique("filter", cniChain, ruleEstablished...); err != nil {
        return fmt.Errorf("failed to add established connection rule: %w", err)
    }

    ruleVxlanToBridge := []string{"-i", vxlanName, "-o", bridgeName, "-j", "ACCEPT"}
    if err := ipt.AppendUnique("filter", cniChain, ruleVxlanToBridge...); err != nil {
        return fmt.Errorf("failed to add vxlan to bridge rule: %w", err)
    }

    ruleBridgeToAny := []string{"-i", bridgeName, "-j", "ACCEPT"}
    if err := ipt.AppendUnique("filter", cniChain, ruleBridgeToAny...); err != nil {
        return fmt.Errorf("failed to add bridge to any rule: %w", err)
    }


    err = ipt.AppendUnique("mangle", "POSTROUTING",
        "-s", localPodCIDR, "-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
        "-j", "TCPMSS", "--set-mss", "1410")
    if err != nil {
        log.Printf("[WARN] Failed to add TCPMSS clamp rule: %v", err)
    }

    err = ipt.AppendUnique("nat", "POSTROUTING",
        "-s", localPodCIDR, "!", "-d", clusterCIDR,
        "-j", "MASQUERADE")
    if err != nil {
        return fmt.Errorf("failed to add masquerade rule: %w", err)
    }

    log.Println("[INFO] Successfully enabled NAT and robust forwarding rules.")
    return nil
}
