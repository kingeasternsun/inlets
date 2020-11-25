package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/kingeastern/inlets/client"
	"github.com/kingeastern/inlets/servermc"
	"github.com/kingeastern/inlets/types"
	"github.com/kingeastern/inlets/util"
)

func Handler(w http.ResponseWriter, req *http.Request) {
	// util.SetReportCaller(true)
	util.Debug("enter")

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	a := struct {
		Name string
		Age  int
	}{
		Name: "kingeasternsunkingeasternsunkingeasternsun",
		Age:  123,
	}

	var err error

	data, _ := json.Marshal(a)

	_, err = w.Write(data)
	if err != nil {
		util.Error(err)
	}
	return
}

func main() {
	serv := flag.Bool("serv", true, "if is server")
	servPort := flag.Int("port", 21002, "server port")
	tokep := flag.String("t", "xxx", "token")
	flag.Parse()

	util.Info(*serv)
	token := *tokep

	//ws的server端
	if *serv {
		cfg := servermc.Server{
			Port:            12001,            //ws的监听端口
			GatewayTimeout:  10 * time.Second, //等待回复的超时时间
			CollectInterval: 10 * time.Second, //等待回复的超时时间
			Token:           token,
		}

		// outgoing := make(chan *http.Request)
		pStr := strconv.Itoa(cfg.Port)
		bus := types.NewBus()
		outgoing := types.NewOnGo()

		go servermc.GarbageCollectBus(bus, cfg.CollectInterval, cfg.GatewayTimeout*2)

		http.HandleFunc("/controller/", servermc.ProxyHandler(outgoing, bus, cfg.GatewayTimeout))
		http.HandleFunc("/tunnel", servermc.ServerWs(outgoing, bus, cfg.Token))

		s := &http.Server{
			Addr:         ":" + pStr,
			ReadTimeout:  20 * time.Second, //超时后，handler会停止接受新的数据，但不会向客户端报错，仍返回200
			WriteTimeout: 20 * time.Second, //超时后，handler会停止接受新的数据，但不会向客户端报错，仍返回200
		}
		util.Error(s.ListenAndServe())

	} else { //ws的客户端

		if *servPort == 0 {
			*servPort = 12002 //所在server的监听地址
		}

		var reConDur time.Duration = 2 //s

		pStr := strconv.Itoa(*servPort)
		go func() {

			upstreamMap := make(map[string]string)
			upstreamMap[""] = "http://127.0.0.1:" + pStr
			client := client.Client{
				Remote:      "127.0.0.1:12001", //ws服务端地址
				UpstreamMap: upstreamMap,
				Token:       token,
			}

			for {
				err := client.Connect()
				if err != nil {
					util.Error(err)
				}

				time.Sleep(reConDur * time.Second)
			}

		}()

		http.HandleFunc("/", Handler)

		s := &http.Server{
			Addr:         ":" + pStr,
			ReadTimeout:  20 * time.Second, //超时后，handler会停止接受新的数据，但不会向客户端报错，仍返回200
			WriteTimeout: 20 * time.Second, //超时后，handler会停止接受新的数据，但不会向客户端报错，仍返回200
		}
		util.Error(s.ListenAndServe())
	}

}

/*
func main() {

	ongo := types.NewOnGo()
	clientID := "clientID"
	client := ongo.Subscribe("clientID", "ws.RemoteAddr().String()")

	defer func() {
		ongo.Unsubscribe(clientID)
	}()

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {

		defer wg.Done()

		for {
			util.Debugf("wait for request %+v", clientID)
			outboundRequest, ok := <-client.Data

			if outboundRequest == nil {
				util.Debug("ok", ok)
				util.Error("close")
				// return
				continue
			}

			util.Debugf("[%s] request written to websocket", outboundRequest.Header.Get(transport.InletsHeader))

		}

	}()

	wg.Wait()

}
*/
// token=$(head -c 16 /dev/urandom | shasum | cut -d" " -f1);
