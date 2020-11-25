package main

import (
	"flag"

	"github.com/BurntSushi/toml"
	"github.com/kingeastern/inlets/types"
	"github.com/kingeastern/inlets/util"
	"github.com/pkg/errors"
)

var sysConfig struct {
	HTTPListenAddr string // 监听端口号
	GroupPre       string //HTTP api 的公共前缀,默认 inlet
	Token          string //
}

var cLientInfo *types.OnGO

func initConfig() error {

	configFilePath := flag.String("c", "./output/inletserver.toml", "configure file path")
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

	if sysConfig.GroupPre == "" {
		sysConfig.GroupPre = "/inlet"
	}

	util.Debugf("get config %+v", sysConfig)

	return nil
}
