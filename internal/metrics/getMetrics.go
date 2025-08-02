package metrics

import (
	"log"
	"net/http"
	"os"
	"rbati-cni/internal/types"
	"strconv"
	"time"

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

	go func ()  {
		for {
			data, err := os.ReadFile(types.COUNT_FILEPATH)
			if err != nil {
				log.Printf("[ERROR] Couldn't read from metric file: %v", err)
			}
			count, err := strconv.Atoi(string(data))
			if err != nil {
				log.Printf("[ERROR] Couldn't convert metric value: %v", err)
			}
			AllocatedIPs.Set(float64(count))
			time.Sleep(5 * time.Second)
		}
	} ()
	
	go func ()  {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9090", nil)
	}()
	

}