/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-18 03:43:40
 * @modify date 2018-01-18 03:43:40
 * @desc [description]
 */
package main

import (
	"flag"
	"net/http"

	_ "net/http/pprof"

	"github.com/lzhig/rapidgo/base"
)

var gApp = &App{}

func main() {
	defer base.LogFlush()
	defer gApp.End()

	defer base.LogPanic()

	flag.Parse()

	go func() {
		http.ListenAndServe(":6061", nil)
	}()

	base.LogInit("./log")

	if err := gApp.Init(); err != nil {
		base.LogError(err)
		return
	}
	gApp.Start()
}
