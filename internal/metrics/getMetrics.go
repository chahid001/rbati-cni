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

	TotalIPs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-ipam_total_ips",
		Help: "Number of total IP available by Subnet.",
	})

	AddOperationsSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-add_operations_success",
		Help: "Number of ADD Operations performed by rbati CNI",
	})

	DelOperationsSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-del_operations_success",
		Help: "Number of DEL Operations performed by rbati CNI",
	})

	AddOperationsFailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-add_operations_failed",
		Help: "Number of ADD Operations performed by rbati CNI",
	})

	DelOperationsFailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rbati_cni-del_operations_failed",
		Help: "Number of DEL Operations performed by rbati CNI",
	})
)

func Init(port string) {

	prometheus.MustRegister(AllocatedIPs,
		TotalIPs,
		AddOperationsSuccess, 
		DelOperationsSuccess,
		AddOperationsFailed,
		DelOperationsFailed)

	go func ()  {
		for {
			data, err := os.ReadFile(types.METRICS_FILEPATH)
			if err != nil {
				log.Printf("[ERROR] Couldn't read from metric file: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			var metrics types.MetricsSnapshot
			if err := json.Unmarshal(data, &metrics); err != nil {
				log.Printf("[ERROR] Couldn't unmarshal metrics: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			AllocatedIPs.Set(float64(metrics.AllocatedIPs))
			TotalIPs.Set(float64(metrics.TotalIPs))
			AddOperationsSuccess.Set(float64(metrics.AddOperationsSuccess))
			DelOperationsSuccess.Set(float64(metrics.DelOperationsSuccess))
			AddOperationsFailed.Set(float64(metrics.AddOperationsFailed))
			DelOperationsFailed.Set(float64(metrics.DelOperationsFailed))
			time.Sleep(5 * time.Second)
		}
	} ()
	
	go func ()  {
		http.Handle("/metrics", promhttp.Handler())
		portStr := ":" + port
		
		if err := http.ListenAndServe(portStr, nil); err != nil {
			log.Printf("[ERROR] Couldn't Start metric server: %v", err)
		}
	}()

}