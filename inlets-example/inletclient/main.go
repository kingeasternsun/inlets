package main

import (
	"os"

	"github.com/kingeastern/inlets/util"
)

func main() {

	util.LoggerToStdoutInit()

	if err := initConfig(); err != nil {
		os.Exit(-1)
	}

	go util.SignalHandle()

	startWebServer()
}
