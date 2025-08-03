package types

//----------------- IPAM --------------------

type LeaseInfo struct {
    TimeStamp   int64 
    ClientID    string
}

//------------ CNI Error ---------------------

type CNIError struct {
	CNIVersion	string `json:"cniVersion"`
    Code       int    `json:"code"`
    Msg        string `json:"msg"`
    Details    string `json:"details,omitempty"`
}


//------------ CNI Result ---------------------

type CNIResult struct {
    CNIVersion  string      `json:"cniVersion"`
    Interfaces  []Interface `json:"interfaces"`
    IPs         []IPConfig  `json:"ips"`
    Routes      []Route     `json:"routes,omitempty"` 
    DNS         *DNSConfig  `json:"dns,omitempty"`    
}


//------------- Generic Types -----------------

type Interface struct {
    Name        string       `json:"name"`
    MAC         string       `json:"mac"`
    Sandbox     string       `json:"sandbox,omitempty"` 
}

type IPConfig struct {
    Version     string       `json:"version"`
    Address     string       `json:"address"` 
    Interface   int          `json:"interface"` 
}

type Route struct {
    Dst         string       `json:"dst"`
    GW          string       `json:"gw,omitempty"`
}

type DNSConfig struct {
    Nameservers []string     `json:"nameservers,omitempty"`
    Domain      string       `json:"domain,omitempty"`
    Search      []string     `json:"search,omitempty"`
    Options     []string     `json:"options,omitempty"`
}

//------------- Metrics Types -----------------

type MetricsSnapshot struct {
    
    AllocatedIPs            int       `json:"allocated_ips"`
    TotalIPs                int       `json:"total_ips"`
    
    AddOperationsSuccess    int       `json:"add_operations_success"`
    AddOperationsFailed     int       `json:"add_operations_failed"`
    
    DelOperationsSuccess    int       `json:"del_operations_success"`
    DelOperationsFailed     int       `json:"del_operations_failed"`

}