package main

import (
  "net"
  "time"
  "log"
  "net/http"
  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "github.com/digitalocean/go-libvirt"
)


var (
  cntCPUDesc = prometheus.NewDesc(
    "vm_cpu",
    "vm cpu",
    []string{"vm"}, nil,
  )
  virt *libvirt.Libvirt
  cache map[string]uint64
  lastTime time.Time
)

type myCollector struct {
}

func (c myCollector) Describe(ch chan<- *prometheus.Desc) {
  prometheus.DescribeByCollect(c, ch)
}

func (c myCollector) Collect(ch chan<- prometheus.Metric) {
  domains, err := virt.Domains()
  if err != nil {
    log.Println(err)
    return
  }
  now := time.Now()
  timeDiff := now.Sub(lastTime)
  lastTime = now
  for _, domain := range(domains) {
    _, _, _, _, cputime, err := virt.DomainGetInfo(domain)
    if err != nil {
      log.Println(err)
      continue
    }
    if cputime == 0 {
      continue
    }
    last, ok := cache[domain.Name]
    if !ok {
      cache[domain.Name] = cputime
      continue
    }
    diff := float64(cputime - last)/1000000000
    diff = diff/timeDiff.Seconds()
    cache[domain.Name] = cputime
    ch <- prometheus.MustNewConstMetric(cntCPUDesc, prometheus.CounterValue, float64(diff), domain.Name)
    
  }

}


func main() {
  conn, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 2*time.Second)
	if err != nil {
		log.Fatalf("failed to dial libvirt: %v", err)
  }
  virt = libvirt.New(conn)
	if err := virt.Connect(); err != nil {
		log.Fatalf("failed to connect: %v", err)
  }
  
  version, err := virt.Version()
	if err != nil {
		log.Fatalf("failed to retrieve libvirt version: %v", err)
	}
  log.Println("libvirt ersion:", version)
  
  cache = map[string]uint64{}
  
  col := myCollector{}
  prometheus.Register(col)
  http.Handle("/metrics", promhttp.Handler())
  http.ListenAndServe(":8081", nil)
}