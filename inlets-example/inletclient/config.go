package main

import (
	"flag"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/kingeastern/inlets/util"
	"github.com/pkg/errors"
)

var sysConfig struct {
	HTTPListenAddr   string // 监听端口号
	InletRegisterURL string // inlet server 注册地址
	Token            string
	ReConnectionDur  time.Duration //重连 inlet server 的时间间隔 单位秒
}

func initConfig() error {

	configFilePath := flag.String("c", "./output/inletclient.toml", "configure file path")
	listenAddr := flag.String("p", "8080", "http listen addr")
	flag.Parse()
	util.Info("get config path :", *configFilePath)
	tomlFilePath := *configFilePath
	if _, err := toml.DecodeFile(tomlFilePath, &sysConfig); err != nil {
		err = errors.Wrap(err, "load toml")
		util.Error(err)
		return err
	}

	// 如果配置文件里面没有配置端口号，就使用默认的启动参数
	if sysConfig.HTTPListenAddr == "" {
		sysConfig.HTTPListenAddr = *listenAddr
	}

	if sysConfig.InletRegisterURL == "" {
		sysConfig.InletRegisterURL = "/inlet"
	}

	util.Debugf("get config %+v", sysConfig)

	return nil
}
