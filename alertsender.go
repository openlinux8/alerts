package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"os"

	"github.com/kinvin/alertsender/pkg/alert"
	"github.com/kinvin/alertsender/pkg/config"
	"github.com/kinvin/alertsender/pkg/log"
	"github.com/kinvin/alertsender/pkg/redis"
	"github.com/kinvin/alertsender/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cfgfile = flag.String("c", "/etc/alertsender/alertsender.yaml", "alert config file")
	logfile = flag.String("l", "/var/log/alertsender.log", "alert log file")
)

var AlertHttpRequestCount = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "alert_http_request_count",
		Help: "Alert http request count"},
)

func init() {
	prometheus.MustRegister(AlertHttpRequestCount)
	prometheus.MustRegister(alert.AlertSendCount)
	prometheus.MustRegister(alert.AlertInhibitCount)
	prometheus.MustRegister(alert.AlertConnRedisFailCount)
	prometheus.MustRegister(utils.AlertSendHttpSuccessCount)
	prometheus.MustRegister(utils.AlertSendHttpFailureCount)
}

func main() {
	flag.Parse()
	f, err := os.Stat(*cfgfile)
	if err != nil {
		stdlog.Fatal(err)
	}
	if f.IsDir() == true {
		stdlog.Fatal("config file is a dir")
	}
	config.InitConfig(*cfgfile)
	log.InitLogger(*logfile)
	err = redis.InitRedis()
	if err != nil {
		fmt.Println("Init Redis Error", err)
	}
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/alert", alertsender)
	err = http.ListenAndServe(":8001", nil)
	if err != nil {
		stdlog.Fatal("ListenAndServe:", err)
	}
}

func alertsender(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		stdlog.Fatal(err)
	}
	defer r.Body.Close()
	AlertHttpRequestCount.Inc()
	alert.HandleAlert(body)
}
