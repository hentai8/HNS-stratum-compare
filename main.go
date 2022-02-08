package main

import (
	"HNS-stratum-compare/cmd/hns"
	"HNS-stratum-compare/configs"
	"HNS-stratum-compare/tools/system"
	"fmt"
	"github.com/jinzhu/configor"
)

var cfg configs.Config

//
// main
// @Description: 程序入口
//
func main() {
	err := configor.Load(&cfg, "configs/config.json")

	//var addrs []string
	//addrs = append(addrs, cfg.Redis.Addr)
	//client := redis.NewClusterClient(&redis.ClusterOptions{
	//    Addrs:    addrs,
	//    Password: cfg.Redis.Password, // no password set
	//})

	if err != nil {
		fmt.Println("read config err=", err)
		return
	}
	for _, c := range cfg.Coins {
		if c.Name == "ltc" {
			var h hns.Hns
			h.StartPoolWatcher(c)
		}
	}
	system.Quit()
}
