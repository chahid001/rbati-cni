package discovery


var supported_versions = []string{"0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0"}


func CheckVersions(Version string) bool {

	for _, sup_v := range supported_versions {

		if Version == sup_v {
			return true 
		}

	}

	return false

}

func Versions() {



}