package network

import (
	"log"

	"github.com/coreos/go-iptables/iptables"
)

func EnableNat(subnet, bridgeName string) error {
    ipt, err := iptables.New()
    if err != nil {
		log.Printf("[ERROR] NAT Error : %v", err)
		return err
    }

    err = ipt.AppendUnique("nat", "POSTROUTING", "-s", subnet, "!", "-o", bridgeName, "-j", "MASQUERADE")
    if err != nil {
		log.Printf("[ERROR] NAT Error : %v", err)
		return err
    }


    err = ipt.AppendUnique("filter", "FORWARD", "-i", bridgeName, "-j", "ACCEPT")
    if err != nil {
		log.Printf("[ERROR] NAT Error : %v", err)
		return err
    }

    err = ipt.AppendUnique("filter", "FORWARD", "-o", bridgeName, "-j", "ACCEPT")
    if err != nil {
		log.Printf("[ERROR] NAT Error : %v", err)
        return err
    }

    log.Println("[INFO] Successfully enabled NAT and forwarding rules.")
    return nil
}