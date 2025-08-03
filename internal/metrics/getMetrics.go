package metrics

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"rbati-cni/internal/types"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (

	AllocatedIPs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-ipam_allocated_ips",
		Help: "Number of current IP allocated by IPAM in BOLTDB.",
	})

	AddOperations = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-add_operations",
		Help: "Number of ADD Operations performed by rbati CNI",
	})

	DelOperations = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-del_operations",
		Help: "Number of DEL Operations performed by rbati CNI",
	})
)

func Init() {

	prometheus.MustRegister(AllocatedIPs, AddOperations, DelOperations)

	go func ()  {
		for {
			data, err := os.ReadFile(types.METRICS_FILEPATH)
			if err != nil {
				log.Printf("[ERROR] Couldn't read from metric file: %v", err)
			}

			var metrics types.MetricsSnapshot
			if err := json.Unmarshal(data, &metrics); err != nil {
				log.Printf("[ERROR] Couldn't unmarshall metrics: %v", err)
			}

			AllocatedIPs.Set(float64(metrics.AllocatedIPs))
			AddOperations.Set(float64(metrics.AddOperations))
			DelOperations.Set(float64(metrics.DelOperations))
			time.Sleep(5 * time.Second)
		}
	} ()
	
	go func ()  {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":9090", nil)
	}()
	

}