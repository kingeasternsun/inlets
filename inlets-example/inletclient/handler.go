package main

import (
	"context"
	"strings"

	"net/http"
	"os"
	"os/signal"
	"time"

	// "vt/weichat"

	// meeting "vt/entranceguard"
	// "vt/log"

	// staticfile "vt/staticfile"

	"github.com/gin-gonic/gin"
	"github.com/kingeastern/inlets/client"
	"github.com/kingeastern/inlets/util"
	"github.com/labstack/gommon/log"
	// "gopkg.in/go-playground/validator.v9"
	// "github.com/go-playground/validator/v10"
)

const MaxFileSize = 100 << 20 //64MB

var MaxExpiredTime = 1 // minute

// fileDB staticfile.FileInfoDB
func genHandler() *gin.Engine {

	router := gin.Default()

	router.MaxMultipartMemory = MaxFileSize
	v2 := router.Group("/inlet/cmd")
	{
		v2.POST("/echo", Echo)
	}

	router.MaxMultipartMemory = MaxFileSize

	return router
}

func startWebServer() {

	go func() {

		var host string
		r := strings.Index(sysConfig.InletRegisterURL, "://")
		if r > 0 {
			host = sysConfig.InletRegisterURL[r+3:]
		} else {
			host = sysConfig.InletRegisterURL
		}

		if strings.Contains(sysConfig.InletRegisterURL, "https") {
			host = "wss://" + host
		} else {
			host = "ws://" + host
		}

		upstreamMap := make(map[string]string)
		upstreamMap[""] = "http://127.0.0.1:" + sysConfig.HTTPListenAddr
		client := client.Client{
			Remote:      host, //ws服务端地址
			UpstreamMap: upstreamMap,
			Token:       sysConfig.Token,
		}

		util.Debugf("client %+v", host)

		for {
			err := client.Connect()
			if err != nil {
				util.Error(err)
			}

			time.Sleep(sysConfig.ReConnectionDur * time.Second)
		}
	}()

	router := genHandler() //单独分割处理，方便用于对handler进行go test测试

	s := &http.Server{
		Handler:      router,
		Addr:         ":" + sysConfig.HTTPListenAddr,
		ReadTimeout:  100 * time.Second,
		WriteTimeout: 100 * time.Second,
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

	err := s.ListenAndServe()
	if err != nil {
		util.Error(err)
	}
	return

}

func HandleGinResponse(c *gin.Context, errRet util.InletError, resInfo interface{}) {
	if errRet.Ret == 0 {
		c.PureJSON(http.StatusOK, resInfo)
	} else {
		var errInfo struct {
			Status int    `json:"status"`
			Msg    string `json:"msg"`
		}

		errInfo.Status = errRet.Ret
		errInfo.Msg = errRet.Msg
		if errRet.Msg == "" {
			errInfo.Msg = errRet.Err.Error()
		}
		c.PureJSON(http.StatusOK, errInfo)
	}
}

// Echo
func Echo(c *gin.Context) {

	var (
		blErr  util.InletError
		reqStu struct {
			Cmd string      `json:"cmd,omitempty"`
			Val interface{} `json:"val,omitempty"`
		}
	)

	defer func() {
		HandleGinResponse(c, blErr, reqStu)
	}()

	if err := c.ShouldBindJSON(&reqStu); err != nil {
		blErr.Ret = util.ErrReq
		blErr.Err = err
		log.Error(blErr)
		return
	}

	util.Debugf("got %+v", reqStu)

	return
}
