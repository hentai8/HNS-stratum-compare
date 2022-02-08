package Logs

import (
	"fmt"
	"log"
	"os"
)

// 存储日志到mysql数据库里
// 存储数据包括延迟，具体触发次数，频率等等

type Log struct {
}

//
// Log
//  @Description: 日志处理
//  @receiver a
//  @param v
//
func (a *Log) Log(v ...interface{}) {
	log.Println(v...)
}

//
// CheckError
//  @Description: 错误处理
//  @receiver a
//  @param err
//
func (a *Log) CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
	}
}
