package mcsql

import (
    "HNS-stratum-compare/configs"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "github.com/jinzhu/configor"
    "github.com/jmoiron/sqlx"
    "sync"
)

var sqlxDB *sqlx.DB
var once sync.Once

//
// GetInstance
//  @Description: 创建可用的数据库连接
//  @return *sqlx.DB
//
func GetInstance() *sqlx.DB {
    once.Do(func() {
        sqlxDB = create()
    })

    return sqlxDB
}

//
// Close
//  @Description: 关闭数据库连接
//
func Close() {
    if sqlxDB != nil {
        sqlxDB.Close()
    }
}

type Conf struct {
    Mysql struct {
        MysqlIp   string `yaml:"MysqlIp"`
        MySqlPort int    `yaml:"MySqlPort"`
        MySqlUser string `yaml:"MySqlUser"`
        MySqlPwd  string `yaml:"MySqlPwd"`
        MySqlTab  string `yaml:"MySqlTab"`
    } `yaml:"Mysql"`
}

var cfg configs.Config

//
// create
//  @Description: 读取配置文件，连接数据库
//  @return *sqlx.DB
//
func create() *sqlx.DB {
    err := configor.Load(&cfg, "../../configs/config.json")
    if err != nil {
        fmt.Println("read config err=", err)
        return nil
    }
    dataSourceName := fmt.Sprintf("%s:%s@%s(%s:%d)/%s",
        cfg.Mysql.Username,
        cfg.Mysql.Password,
        cfg.Mysql.Network,
        cfg.Mysql.Server,
        cfg.Mysql.Port,
        cfg.Mysql.Database)
    database, err := sqlx.Open("mysql", dataSourceName)

    if err != nil {
        fmt.Println("open mysql failed,", err)
        return nil
    }

    return database
}
