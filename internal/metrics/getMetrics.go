package metrics

import (
	"net/http"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (

	AllocatedIPs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-ipam_allocated_ips",
		Help: "Number of current IP allocated by IPAM in BOLTDB.",
	})

)

func Init() {

	prometheus.MustRegister(AllocatedIPs)

	// go func ()  {
	// 	for {
	// 		count := ipamOBJ.CountAllocatedIPs()
	// 		AllocatedIPs.Set(float64(count))
	// 		time.Sleep(5 * time.Second)
	// 	}
	// } ()
	
	go func ()  {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9090", nil)
	}()
	

}