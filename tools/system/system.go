package system

import (
    "log"
    "os"
    "os/signal"
    "syscall"
)

//
// Quit
// @Description: 安全退出
//
func Quit() {
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
    for {
        sig := <-sc
        switch sig {
        case syscall.SIGQUIT, syscall.SIGTERM:
            log.Panic("SIGQUIT")
        case syscall.SIGHUP:
            log.Panic("SIGHUP")
        case syscall.SIGINT:
            log.Panic("SIGINT")
        default:
            break
        }
    }
}
