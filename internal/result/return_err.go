package result

import (
	"encoding/json"
	"fmt"
	"os"
	"rbati-cni/internal/types"
)

func ReturnErr(ErrObject *types.CNIError) {

	errBytes, _ := json.Marshal(ErrObject)

	fmt.Fprintf(os.Stderr, string(errBytes))

}
