package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/kingeastern/inlets/servermc"
	"github.com/kingeastern/inlets/types"
	"github.com/kingeastern/inlets/util"
	"github.com/pkg/errors"
)

func startWebServer() {

	bus := types.NewBus()
	cLientInfo = types.NewOnGo()

	port, err := strconv.Atoi(sysConfig.HTTPListenAddr)
	if err != nil {
		err = errors.Wrap(err, sysConfig.HTTPListenAddr)
		util.Error(err)
		return
	}
	cfg := servermc.Server{
		Port:            port,             //ws的监听端口
		GatewayTimeout:  10 * time.Second, //等待回复的超时时间
		CollectInterval: 10 * time.Second, //等待回复的超时时间
		Token:           sysConfig.Token,
	}

	go servermc.GarbageCollectBus(bus, cfg.CollectInterval, cfg.GatewayTimeout*2)

	http.HandleFunc(sysConfig.GroupPre+"/register", servermc.ServerWs(cLientInfo, bus, cfg.Token))          // inlet client 使用这个接口注册进来
	http.HandleFunc(sysConfig.GroupPre+"/cmd/", servermc.ProxyHandler(cLientInfo, bus, cfg.GatewayTimeout)) //用户通过这个接口往下游的inlet client发送http请求
	http.HandleFunc(sysConfig.GroupPre+"/clients", QueryInletsClientsHandler)                               //获取 inlet client的列表信息

	s := &http.Server{
		Addr:         ":" + sysConfig.HTTPListenAddr,
		ReadTimeout:  50 * time.Second,
		WriteTimeout: 50 * time.Second,
		//WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	//接受信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		defer cancel()
		util.Debug("get singal to stop")
		if err := s.Shutdown(ctx); err != nil {
			util.Errorf("unable to shutdown server: %v", err)
		} else {
			util.Info("gracefuly shut down")
		}
	}()

	err = s.ListenAndServe()
	if err != nil {
		util.Error(err)
	}
	return
}

// InletClient 连接到当前server的inlet 客户端信息
type InletClient struct {
	Created  *time.Time `json:"created,omitempty"`
	Addr     string     `json:"addr,omitempty"`
	ClientID string     `json:"client_id,omitempty"`
}

// QueryInletsClientsHandler 查询
func QueryInletsClientsHandler(w http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodOptions {
		return
	}

	var (
		resInfo struct {
			Msg  string        `json:"msg"`
			Ret  int           `json:"status"`
			Info []InletClient `json:"info"`
		}
		blErr util.InletError
	)

	resInfo.Msg = "ok"
	defer func() {
		util.HandleResponse(w, blErr, resInfo)
	}()
	info := cLientInfo.ClientList()
	resInfo.Info = make([]InletClient, len(info))

	for i := range info {
		resInfo.Info[i].Addr = info[i].Addr
		resInfo.Info[i].ClientID = info[i].ClientID
		resInfo.Info[i].Created = info[i].Created
	}

	return
}
