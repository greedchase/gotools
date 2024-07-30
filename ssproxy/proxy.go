// proxy
package main

import (
	"github.com/greedchase/gotools/stlog"
	"github.com/greedchase/gotools/stnet"
)

var (
	LOG   = stlog.NewFileLoggerWithoutTerm("proxy.log")
	proxy *stnet.Server
)

func Init() error {
	e := LoadCfg()
	if e != nil {
		return e
	}
	proxy = stnet.NewServer(10, 64)
	e = AddLCProxy(proxy, listenConnect)
	if e != nil {
		return e
	}
	e = AddLLProxy(proxy, listenListen)
	if e != nil {
		return e
	}
	e = AddCCProxy(proxy, connectConnect)
	if e != nil {
		return e
	}

	return nil
}
