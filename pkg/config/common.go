package config

import (
	"time"
)

type Alert struct {
	Status            string            `json:"status"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	StartsAt          time.Time         `json:"startsAt"`
	EndsAt            time.Time         `json:"endsAt"`
	FingerPrint       string            `json:"fingerprint"`
	FiringSendCount   int               `json:"-"`
	InhibitionCount   int               `json:"-"`
	ResolvedSendCount int               `json:"-"`
	SendTime          int64             `json:"-"`
}

type Notification struct {
	Status string  `json:"status"`
	Alerts []Alert `json:"alerts"`
}

type Config struct {
	EnableDingDing      bool                `yaml:"enableDingDing"`
	EnableWeChat        bool                `yaml:"enableWeChat"`
	AdminDingDingTokens []map[string]string `yaml:"adminDingDingTokens,omitempty"`
	AdminWeChatTokens   []map[string]string `yaml:"adminWeChatTokens,omitempty"`
	Namespaces          map[string]string   `yaml:"namespaces,omitempty"`
	Redis               Redis               `yaml:"redis,omitempty"`
	Receivers           []*Receiver         `yaml:"receivers,omitempty"`
}

type Redis struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}

type Receiver struct {
	Name           string              `yaml:"name,omitempty"`
	DingDingTokens []map[string]string `yaml:"dingDingTokens,omitempty"`
	WeChatTokens   []map[string]string `yaml:"weChatTokens,omitempty"`
}
