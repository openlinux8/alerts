package alert

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/kinvin/alertsender/pkg/config"
	zaplog "github.com/kinvin/alertsender/pkg/log"
	"github.com/kinvin/alertsender/pkg/redis"
	"github.com/kinvin/alertsender/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	dingdingRobotUrl string = "https://api.dingtalk.com/robot/send?access_token="
	wechatRobotUrl   string = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="
)

var (
	logger *zap.SugaredLogger
)

var AlertSendCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "alert_send_count",
		Help: "Firing And Resolved alert send count"},
	[]string{"alertname", "attr", "finger", "namespace", "status"},
)

var AlertInhibitCount = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "alert_inhibit_count",
		Help: "Inhibit alert send count"},
	[]string{"alertname", "attr", "finger", "namespace", "status"},
)

var AlertConnRedisFailCount = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "alert_conn_redis_fail_count",
		Help: "Redis connection failure count"},
)

func HandleAlert(body []byte) {
	logger = zaplog.GetLogger()
	configs := config.GetConfig()
	noti := new(config.Notification)
	err := json.Unmarshal(body, noti)
	if err != nil {
		fmt.Printf("Can't Unmarshal Json: %s\n", err)
		return
	}

	receiver := "admin"
	DingDingTokens := configs.AdminDingDingTokens
	WeChatTokens := configs.AdminWeChatTokens
	timenow := time.Now()
	if noti.Status == "firing" {
		noti.Status = "警报"
	} else if noti.Status == "resolved" {
		noti.Status = "恢复"
	}
	for _, v := range noti.Alerts {
		alertAlertname := v.Labels["alertname"]
		alertSeverity := v.Labels["severity"]
		alertNamespace := v.Labels["namespace"]
		alertDescription := v.Annotations["description"]
		alertStartAt := v.StartsAt.Format("2006-01-02 15:04:05")
		v.SendTime = v.StartsAt.Unix()
		flag, err := redis.RedisAlertHandle(timenow, &v)
		if err != nil {
			AlertConnRedisFailCount.Inc()
			flag = true
			logger.Errorf("RedisAlertHandle Error: %s", err)
		}
		if flag {
			AlertSendCount.WithLabelValues(alertAlertname, v.Labels["attr"], v.FingerPrint, alertNamespace, v.Status).Inc()
			AlertJsonstr := fmt.Sprintf("状态: %v\n告警名: %v\n告警级别: %v\n命名空间: %v\n开始时间: %v\n描述: %v\n",
				noti.Status, alertAlertname, alertSeverity,
				alertNamespace, alertStartAt, alertDescription)
			for k, v := range configs.Namespaces {
				if alertNamespace == k {
					for _, r := range configs.Receivers {
						if v == r.Name {
							receiver = r.Name
							DingDingTokens = r.DingDingTokens
							WeChatTokens = r.WeChatTokens
							break
						}
					}
					break
				}
			}
			if configs.EnableDingDing {
				alertChannel(receiver, "dingding", AlertJsonstr, DingDingTokens)
			}
			if configs.EnableWeChat {
				alertChannel(receiver, "wechat", AlertJsonstr, WeChatTokens)
			}
		} else {
			AlertInhibitCount.WithLabelValues(alertAlertname, v.Labels["attr"], v.FingerPrint, alertNamespace, v.Status).Inc()
		}
	}
}

func alertChannel(receiver, alertType, jsonstr string, tokens []map[string]string) {
	tokenNum := len(tokens)
	if tokenNum != 0 {
		for _, v := range tokens {
			flag, err := redis.RedisTokenHandle(receiver, v["token"])
			if err != nil {
				AlertConnRedisFailCount.Inc()
				flag = true
				logger.Errorf("RedisTokenHandle Error: %s", err)
			}
			if flag {
				sendAlert(jsonstr, alertType, v)
				return
			}
		}
		logger.Errorf("所有Token发送次数都达到20次，receiver: %s, alertType: %s, info: %s", receiver, alertType, jsonstr)
	}
}

func sign(secret string) (signstr string) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	message := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	ec := base64.StdEncoding.EncodeToString(h.Sum(nil))
	signstr = fmt.Sprintf("&timestamp=%s&sign=%s", timestamp, ec)
	return
}

func sendAlert(jsonstr, alertType string, token map[string]string) {
	var url string
	if alertType == "dingding" {
		if secret, ok := token["secret"]; ok {
			signstr := sign(secret)
			url = dingdingRobotUrl + token["token"] + signstr
		} else {
			url = dingdingRobotUrl + token["token"]
		}
	} else {
		url = wechatRobotUrl + token["token"]
	}
	alertJsonstr := fmt.Sprintf(`{"msgtype":"text","text":{"content":"%s"}}`, jsonstr)
	utils.HandleHttp(url, []byte(alertJsonstr))
}
