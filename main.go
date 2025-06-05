package main

import (
	"ft_cni/pkg/load"
	"ft_cni/pkg/log"
)

var (
	envMap map[string]string
	rawJSON []byte
)

func init() {

	envMap = load.ENV()
	rawJSON = load.RawJson()

}



func main() {

	log.LogCalls(envMap, string(rawJSON))

}


