package network

import (
	"log"

	"github.com/coreos/go-iptables/iptables"
)


func EnableNat(subnet, bridgeName string) error {
    ipt, err := iptables.New()
    if err != nil {
        log.Printf("[ERROR] NAT Error: %v", err)
        return err
    }

    err = ipt.Insert("filter", "FORWARD", 1, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
    if err != nil {
        log.Printf("[ERROR] NAT Error: Failed to add conntrack rule: %v", err)
        return err
    }
    

    podSubnet := subnet
    nodeSubnet := "192.168.56.0/24"

    err = ipt.AppendUnique(
        "nat", "POSTROUTING",
        "-s", podSubnet, "!", "-d", nodeSubnet,
        "-j", "MASQUERADE",
    )
    if err != nil {
        log.Printf("[ERROR] NAT Error: Failed to add MASQUERADE rule: %v", err)
        return err
    }

    err = ipt.AppendUnique("filter", "FORWARD", "-i", bridgeName, "-j", "ACCEPT")
    if err != nil {
        log.Printf("[ERROR] NAT Error: %v", err)
        return err
    }
    err = ipt.AppendUnique("filter", "FORWARD", "-o", bridgeName, "-j", "ACCEPT")
    if err != nil {
        log.Printf("[ERROR] NAT Error: %v", err)
        return err
    }

    log.Println("[INFO] Successfully enabled NAT and forwarding rules.")
    return nil
}