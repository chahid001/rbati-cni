package types

type CNIError struct {
	CNIVersion	string `json:"cniVersion"`
    Code       int    `json:"code"`
    Msg        string `json:"msg"`
    Details    string `json:"details,omitempty"`
}