package utils

import (
	"os"
	"rbati-cni/internal/result"
	"rbati-cni/internal/types"
)



func CreateErr(_CNIVersion , _Msg, _Detail string, _Code int) {

	cniErr := types.CNIError{
		CNIVersion: _CNIVersion,
		Code:       _Code,
		Msg:        _Msg,
		Details:	_Detail,
	}

	result.ReturnErr(&cniErr)

	os.Exit(1)
}
