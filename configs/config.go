package configs

//
// Config
//  @Description: 配置文件
//
type Config struct {
    Mysql Mysql   `json:"mysql"`
    Redis Redis   `json:"redis"`
    Coins []Coins `json:"coins"`
}

//
// Coins
//  @Description: 各币种矿池配置文件
//
type Coins struct {
    Name       string `json:"name"`
    Port       string `json:"port"`
    TargetPool struct {
        Name      string `json:"name"`
        Host      string `json:"host"`
        Port      string `json:"port"`
        Worker    string `json:"worker"`
        Enable    bool   `json:"enable"`
        ReportUrl string `json:"report_url"`
    } `json:"target_pool"`
    ListeningPools []struct {
        Name   string `json:"name"`
        Host   string `json:"host"`
        Port   string `json:"port"`
        Worker string `json:"worker"`
        Enable bool   `json:"enable"`
    } `json:"listening_pools"`
}

//
// Mysql
//  @Description: 数据库配置文件
//
type Mysql struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Network  string `json:"network"`
    Server   string `json:"server"`
    Port     int    `json:"port"`
    Database string `json:"database"`
}

type Redis struct {
    Addr     string `json:"addr"`
    Password string `json:"password"`
}
