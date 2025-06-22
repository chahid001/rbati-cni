package result

import (
	"encoding/json"
	"fmt"
	"ft_cni/internal/types"
	"os"
)

func ReturnErr(ErrObject *types.CNIError) {

	errBytes, _ := json.Marshal(ErrObject)

	fmt.Fprintf(os.Stderr, string(errBytes))

}
