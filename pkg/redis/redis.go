package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kinvin/alertsender/pkg/config"
	"github.com/kinvin/alertsender/pkg/utils"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

func InitRedis() (err error) {
	configs := config.GetConfig()
	redisAddr := fmt.Sprintf("%s:%d", configs.Redis.Host, configs.Redis.Port)
	redisPass := configs.Redis.Password
	rdb = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPass,
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 5,
	})
	_, err = rdb.Ping(ctx).Result()
	return
}

//返回的flag为true，将触发警报，flag为false将限制本条警报
func RedisAlertHandle(timenow time.Time, alert *config.Alert) (flag bool, err error) {
	flag = true
	//查询Redis是否存在当前警报的记录
	status, sendtime, firingsendcount, err := GetAlertKey(alert)
	if err != nil {
		return
	}
	// status为空，说明没有这条警报记录
	if status == "" || status != alert.Status {
		err = SetAlertKey(alert, alert.SendTime, false)
		return
	}
	if status == "firing" {
		inhibition := utils.RestTime(timenow, alert.StartsAt, sendtime, alert.Labels["severity"], firingsendcount)
		if !inhibition {
			sendtime = timenow.Unix()
		}
		err = SetAlertKey(alert, sendtime, inhibition)
		flag = !inhibition
	} else if status == "resolved" {
		err = SetAlertKey(alert, sendtime, false)
		flag = false
	}
	return
}

func GetAlertKey(alert *config.Alert) (status string, sendtime int64, firingsendcount int, err error) {
	key := fmt.Sprintf("%s:%s:%s", alert.Labels["alertname"], alert.Labels["attr"], alert.FingerPrint)
	flag, err := rdb.HExists(ctx, key, "status").Result()
	if err == nil && flag {
		status, err = rdb.HGet(ctx, key, "status").Result()
		sendtime, err = rdb.HGet(ctx, key, "sendtime").Int64()
		firingsendcount, err = rdb.HGet(ctx, key, "firingsendcount").Int()
	}
	return
}

func SetAlertKey(alert *config.Alert, sendtime int64, inhibition bool) (err error) {
	var field string
	key := fmt.Sprintf("%s:%s:%s", alert.Labels["alertname"], alert.Labels["attr"], alert.FingerPrint)
	if inhibition {
		field = "inhibitionCount"
	} else {
		field = alert.Status + "sendcount"
	}
	_, err = rdb.HSet(ctx, key, "status", alert.Status, "labels.namespace", alert.Labels["namespace"], "sendtime", sendtime).Result()
	if err != nil {
		return
	}
	_, err = rdb.HIncrBy(ctx, key, field, 1).Result()
	return
}

func RedisTokenHandle(receiver, token string) (flag bool, err error) {
	key := fmt.Sprintf("%s:%s", receiver, token)
	tokens, err := rdb.Keys(ctx, key+"*").Result()
	if err != nil {
		return
	}
	// 微信和钉钉群都限制每个机器人每分钟只能发送20条信息
	num := len(tokens)
	if num >= 20 {
		return
	}
	newkey := fmt.Sprintf("%s:%d", key, num)
	_, err = rdb.Set(ctx, newkey, 1, 60*time.Second).Result()
	if err != nil {
		return
	}
	return true, nil
}
