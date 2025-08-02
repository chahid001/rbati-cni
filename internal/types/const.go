package types


const (

	BUCKET_NAME 		= "ipam"
	IPAM_FODLER_PATH 	= "/var/lib/cni/rbati-cni"
	IPAM_DB_FILENAME 	= "ipam.db"

	LOG_FILE_PATH 		= "/var/log/rbati-cni"
	LOG_FILE_FILENAME	= "rbati-cni.log"
	
	CONFIG_FILE_PATH	= "/etc/cni/net.d/10-rbati.conf"

	BRIDGE_NAME 		= "rbati.0"
	VXLAN_NAME 			= "rbati.1" 	

	ENV_PATH 			= "/run/rbati-cni"
	ENV_FILENAME 		= "rbati.env"

	MAC_ANNOTATION 		= "cni.rbati.io/vxlan-mac"

	COUNT_FILEPATH		= "/run/rbati-cni/ipamIP"
)