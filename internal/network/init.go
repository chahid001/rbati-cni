package network

import (
	"log"

	"github.com/coreos/go-iptables/iptables"
)

// TODO : change values to parametres
func EnableNat(Subnet, BridgeName string) error {

	ipt, err := iptables.New()


	if err != nil {
		return err
	}

	rule, err := ipt.Exists("nat", "POSTROUTING",
		"-s", Subnet, "!", "-o", BridgeName, "-j", "MASQUERADE")

	if err != nil {
		return err 
	}

	if !rule {
		err := ipt.Append("nat", "POSTROUTING",
		"-s", Subnet, "!", "-o", BridgeName, "-j", "MASQUERADE")

		if err != nil {
			return err
		}
	}

	log.Println("[INFO] Enabling NAT")
	return nil
}