package hns

import (
	"HNS-stratum-compare/configs"
	"HNS-stratum-compare/tools/mysql"
	"HNS-stratum-compare/tools/redis"
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/redis.v5"
	"log"
	"math"
	"net"
	"strconv"
	"time"
)

// 启动对各个矿池的监听 模拟矿工
// 启动对监听到的各个矿池的高度的比对
// createSocket() 与各个矿池建立连接
// onMessage() 监听各个矿池的推送，并且存入redis中

type Hns struct {
	RedisClient *redis.Client
	ctx         context.Context
	ctxCancel   func()
	NetClient   *net.TCPConn
}

type Receive struct {
	Id     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type Notify struct {
	TaskNum       string   `json:"task_num"`
	HashPreBlock  string   `json:"hash_pre_block"`
	Coinbase1     string   `json:"coinbase_1"`
	Coinbase2     string   `json:"coinbase_2"`
	TxIDs         []string `json:"tx_i_ds"`
	NVersion      string   `json:"n_version"`
	NBits         string   `json:"n_bits"`
	NTime         string   `json:"n_time"`
	State         bool     `json:"state"`
	Height        int64    `json:"height"`
	CoinbaseValue string   `json:"coinbase_value"`
}

//
// StartPoolWatcher
//  @Description: 读取配置文件，开启PoolWatcher
//  @receiver l *Hns
//  @param c 配置文件
//
func (l *Hns) StartPoolWatcher(c configs.Coins) {
	//// 初始化api
	//var a api.Api

	// 初始化ctx,ctxCancel
	l.ctx, l.ctxCancel = context.WithCancel(context.Background())

	// 创建管道,负责在监听到可挖空块后,跨越线程,通知给长连接
	ltcData := make(chan interface{})

	// 初始化redis连接
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	l.RedisClient = client

	// 清除原有数据
	keys, err := redisOperation.Keys(l.RedisClient, "ltc:*").Result()
	if err != nil {
		fmt.Println("redis get pre ltc data error=", err.Error())
		return
	}
	for _, key := range keys {
		_, err := redisOperation.Del(l.RedisClient, key).Result()
		if err != nil {
			fmt.Println("redis del pre ltc data error=", err.Error())
			return
		}
	}

	// 针对目标矿池的监控
	targetUrl := c.TargetPool.Host + ":" + c.TargetPool.Port
	fmt.Println(c.Name, " TargetPools url:", targetUrl)
	ctx, cancel := context.WithCancel(context.Background())
	go l.CreateSocket(targetUrl, targetUrl, c.TargetPool.Worker, ltcData, cancel)
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				println("need reconnect")
				ctx, cancel = context.WithCancel(context.Background())
				go l.CreateSocket(targetUrl, targetUrl, c.TargetPool.Worker, ltcData, cancel)
			}
		}
	}(ctx)

	for _, p := range c.ListeningPools {
		url := p.Host + ":" + p.Port
		fmt.Println(c.Name, " ListeningPools url:", url)
		ctx, cancel := context.WithCancel(context.Background())
		go l.CreateSocket(url, targetUrl, c.TargetPool.Worker, ltcData, cancel)
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					println("need reconnect")
					ctx, cancel = context.WithCancel(context.Background())
					go l.CreateSocket(url, targetUrl, c.TargetPool.Worker, ltcData, cancel)
				}
			}
		}(ctx)
	}

	//// 启动通知的长连接
	//reportUrl := c.TargetPool.ReportUrl
	//go a.CreateServer(reportUrl, client, ltcData)
}

//
// CreateSocket
//  @Description: 与其他矿池创立连接
//  @receiver l *Hns
//  @param url 矿池地址
//  @param targetUrl 目标矿池地址
//  @param worker 矿工名
//  @param ltcData 数据传输管道
//  @param cancel 自动重启
//  @return conn tcp长连接
//
func (l *Hns) CreateSocket(url string, targetUrl string, worker string, ltcData chan interface{}, cancel func()) (conn *net.TCPConn) {
	tcpAdd, err := net.ResolveTCPAddr("tcp", url)
	if err != nil {
		fmt.Println("net.ResolveTCPAddr err=", err.Error())
		return
	}
	for {
		if conn, err = net.DialTCP("tcp", nil, tcpAdd); err == nil {
			conn.SetKeepAlive(true)
			fmt.Println("connected:", url)
			go l.OnMessageReceived(conn, url, targetUrl, ltcData, cancel)
			sub := []byte("{\"id\": 1, \"method\": \"mining.subscribe\", \"params\": [\"cpuminer-opt/3.8.8.5-cpu-pool\"]}\n")
			if _, err = conn.Write(sub); err != nil {
				log.Println("client send subscribe failed, err: ", err.Error())
			} else {
				auth := []byte("{\"id\": 2, \"method\": \"mining.authorize\", \"params\": [\"" + worker + "\", \"123\"]}\n")
				if _, err = conn.Write(auth); err != nil {
					log.Println("client send auth failed, err: ", err.Error())
				} else {
					println("success")
					return conn
				}
			}
		} else {
			println("failed to connect to :", url, " try again after 5 seconds")
			time.Sleep(5 * time.Second)
		}
		time.Sleep(5 * time.Second)
	}
}

//
// OnMessageReceived
//  @Description: 监听消息,并进行处理
//  @receiver l *Hns
//  @param conn tcp长连接
//  @param url 矿池地址
//  @param targetUrl 目标矿池地址
//  @param ltcData 数据传输管道
//  @param cancel 自动重启
//
func (l *Hns) OnMessageReceived(conn *net.TCPConn, url string, targetUrl string, ltcData chan interface{}, cancel func()) {
	reader := bufio.NewReader(conn)
	for {
		select {
		default:
			if msg, isPrefix, err := reader.ReadLine(); err != nil {
				println("failed to read line:", err.Error())
				err := conn.Close()
				if err != nil {
					println(err.Error())
				}
				println("success close tcp connection  ", url)
				//l.ctxCancel()
				cancel()
				return
			} else if isPrefix {
				println("buffer size small")
				return
			} else {
				var r Receive
				err = json.Unmarshal([]byte(msg), &r)
				if err != nil {
					fmt.Println("unmarshal pool watcher json error=", err.Error())
					return
				}
				//如果监听到mining.notify信息,则对其进行解码
				if r.Method == "mining.notify" {
					println("receive mining.notify ", url)
					p := r.Params
					var notify Notify
					// 检验数据是否完整
					if len(p) < 9 {
						fmt.Println("This length of mining.notify data is illegal")
						continue
					}
					//var ok bool
					notify.TaskNum = p[0].(string)
					// 此HashPreBlock为经过转换的preHash，发送到stratum端后需要进行再次转换
					notify.HashPreBlock = p[1].(string)
					notify.Coinbase1 = p[2].(string)
					notify.Coinbase2 = p[3].(string)
					//notify.TxIDs = p[4].([]string)
					notify.NVersion = p[5].(string)
					notify.NBits = p[6].(string)
					notify.NTime = p[7].(string)
					notify.State = p[8].(bool)

					// 对高度进行16进制解码
					heightBuff := notify.Coinbase1[84:]
					heightLength, err := hex.DecodeString(heightBuff[0:2])
					if err != nil {
						fmt.Println("decode height error=", err.Error())
					}

					heightLengthInt := int(heightLength[0])
					var height int64
					height = 0
					for i := 0; i < heightLengthInt; i++ {
						heightPlus, err := hex.DecodeString(heightBuff[2+2*i : 4+2*i])
						if err != nil {
							fmt.Println("decode height error=", err.Error())
						}
						height = height + int64(heightPlus[0])*int64(math.Pow(256, float64(i)))
					}
					fmt.Println(url, ":height:", height)

					notify.CoinbaseValue = l.GetCoinbaseValue(height)

					// 将高度存到redis内
					_, err = redisOperation.Set(l.RedisClient, "ltc:"+url, strconv.FormatInt(int64(height), 10)).Result()
					if err != nil {
						fmt.Println("redis set ltc data error=", err.Error())
						return
					}
					notify.Height = height
					jsonNotify, err := json.Marshal(notify)
					if err != nil {
						fmt.Println("marsh notify json error=", err.Error())
						return
					}
					//将具体notify解码后的信息存入redis内
					strNotify := string(jsonNotify)
					//fmt.Println(strNotify)
					_, err = redisOperation.Set(l.RedisClient, "ltcNotify:"+url, strNotify).Result()
					if err != nil {
						fmt.Println("redis set ltcNotify data error=", err.Error())
						return
					}

					//每当更新一次notify信息,就将高度进行一次比较
					fmt.Println("success get notify, url:" + url)
					l.CompareHeight(targetUrl, ltcData)
				}
			}
		}
	}
}

//
// GetCoinbaseValue
//  @Description: 根据高度计算coinbaseValue
//  @receiver l *Hns
//  @param height 高度
//  @return coinbaseValue 出块奖励
//
func (l *Hns) GetCoinbaseValue(height int64) (coinbaseValue string) {
	t := int64(height / 840000)
	if t >= 33 {
		return "0"
	}
	coinbaseValue = strconv.FormatFloat(50/(math.Pow(2, float64(t))), 'G', -1, 64)
	return
}

//
// CompareHeight
//  @Description: 一旦有矿池的高度发生变化,将目标矿池与其他矿池进行比较
//  @receiver l *Hns
//  @param targetUrl 目标矿池地址
//  @param ltcData 数据传输管道
//
func (l *Hns) CompareHeight(targetUrl string, ltcData chan interface{}) {
	keys, err := redisOperation.Keys(l.RedisClient, "ltc:*").Result()
	if err != nil {
		fmt.Println("redis.Do err=", err)
		return
	}
	mainHeight, err := redisOperation.Get(l.RedisClient, "ltc:"+targetUrl).Result()
	if err != nil {
		fmt.Println("fail to get targetUrl Data err=", err)
		return
	}
	fmt.Println(mainHeight)
	mainH, err := strconv.ParseInt(mainHeight, 10, 64)
	if err != nil {
		fmt.Println("strconv.ParseInt err=", err)
		return
	}

	// 遍历ltc的所有url解码出来的内容
	for _, v := range keys {
		if v != "{pool-watcher}:ltc:"+targetUrl {
			height, err := redisOperation.GetByFor(l.RedisClient, v).Result()
			if err != nil {
				fmt.Println("fail to get height Data err=", err)
				return
			}
			h, err := strconv.ParseInt(height, 10, 64)
			if err != nil {
				fmt.Println("strconv.ParseInt err=", err)
				return
			}
			fmt.Println("compare:", v, ":", h)
			fmt.Println("the difference of height", v, ":", h-mainH)
			// 比较高度,如果高度是1,则tell给targetUrl
			if h-mainH == 1 {
				//if true {
				fmt.Println("!.!.!.start mining empty block")
				url := v[19:]
				//go l.Tell(url, ltcData)

				// 记录数据
				// 开始时间,结束时间,高度,url
				// 其中结束时间需要进行计算

				startTime := time.Now().Unix()

				for {
					mainHeightAfter, err := redisOperation.Get(l.RedisClient, "ltc:"+targetUrl).Result()
					if err != nil {
						fmt.Println("fail to get targetUrl Data err=", err)
						return
					}
					fmt.Println(mainHeightAfter)
					mainHAfter, err := strconv.ParseInt(mainHeightAfter, 10, 64)
					if err != nil {
						fmt.Println("strconv.ParseInt err=", err)
						return
					}
					if mainHAfter == h {
						endTime := time.Now().Unix()
						go l.Record(h, url, startTime, endTime)
					}
				}
			}
		}
	}
}

//
// Tell
//  @Description: 将url的数据传入ltcData这个chan中，通知给api模块
//  @receiver l *Hns
//  @param url 矿池地址
//  @param ltcData 数据传输管道
//
func (l *Hns) Tell(url string, ltcData chan interface{}) {
	NotifyData, err := redisOperation.Get(l.RedisClient, "ltcNotify:"+url).Result()
	if err != nil {
		fmt.Println("redis.Do err=", err)
		return
	}
	//fmt.Println("TELL:", NotifyData)
	ltcData <- NotifyData
}

//
// Record
//  @Description: 将空块信息存入数据库中，包括高度、矿池地址、开始时间
//  @receiver l
//  @param height 高度
//  @param url 矿池地址
//
func (l *Hns) Record(height int64, url string, startTime int64, endTime int64) {
	//startTime := time.Now().Unix()
	db := mcsql.GetInstance()

	sql := `insert into ltc (height, url, start_time, end_time) values (?,?,?)`
	_, err := db.Exec(sql, height, url, startTime, endTime)
	if err != nil {
		fmt.Println("record to mysql err=", err)
	}
	return
}
