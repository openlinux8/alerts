package utils

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"time"

	zaplog "github.com/kinvin/alertsender/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	emergency int64 = 1
	critical  int64 = 2
	warning   int64 = 3
	notice    int64 = 4
	info      int64 = 5
)

var (
	logger *zap.SugaredLogger
)

var AlertSendHttpSuccessCount = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "alert_send_http_success_count",
		Help: "Success send alert count"},
)

var AlertSendHttpFailureCount = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "alert_send_http_failure_count",
		Help: "Failure send alert count"},
)

func HandleHttp(alertUrl string, jsonstr []byte) {
	logger = zaplog.GetLogger()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Second * 2}
	req, err := http.NewRequest("POST", alertUrl, bytes.NewBuffer(jsonstr))
	if err != nil {
		AlertSendHttpFailureCount.Inc()
		logger.Errorf("Data post to %s Error: %s\n", alertUrl, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		AlertSendHttpFailureCount.Inc()
		logger.Errorf("Send data to %s Error: %s\n", alertUrl, err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	AlertSendHttpSuccessCount.Inc()
	logger.Infof("Server Url: %s, Server response Body: %s\n", alertUrl, body)
	//logger.Infof("Send Alert Content: %s\n", jsonstr)

}

func RestTime(timenow, startTime time.Time, sendtime int64, severity string, sendCount int) (inhibition bool) {
	var timeout int64
	// 告警超过三天还不处理，将不再发送警报
	if timenow.Unix()-startTime.Unix() > 259200 {
		inhibition = true
		return
	}
	switch severity {
	case "emergency":
		timeout = emergency * 300
	case "critical":
		timeout = critical * 3600
	case "warning":
		timeout = warning * 3600
	case "notice":
		timeout = notice * 3600
	default:
		timeout = info * 3600
	}
	// 警报发送超过三次，发送时间将变为原来的2倍(比如，emergency原来是1小时发送一次，发送三次后，第四次开始两小时发送一次)
	if sendCount >= 3 {
		timeout *= 2
		// 警报发送超过三次，如果是0到8点之间和周六或周日，不会发送。
		t := timenow.Hour()
		weekDay := int(timenow.Weekday())
		if t < 8 || weekDay == 0 || weekDay == 6 {
			inhibition = true
		}
	}
	if timenow.Unix()-sendtime < timeout {
		inhibition = true
	}
	return
}
