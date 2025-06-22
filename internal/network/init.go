package network

import (
	"github.com/coreos/go-iptables/iptables"
)


func EnableNat() error {

	ipt, err := iptables.New()


	if err != nil {
		return err
	}

	rule, err := ipt.Exists("nat", "POSTROUTING",
		"-s", "10.42.0.0/24", "!", "-o", "cni_br0", "-j", "MASQUERADE")

	if err != nil {
		return err 
	}

	if !rule {
		err := ipt.Append("nat", "POSTROUTING",
		"-s", "10.42.0.0/24", "!", "-o", "cni_br0", "-j", "MASQUERADE")

		if err != nil {
			return err
		}
	}

	return nil
}