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
	"log"
	"net/http"

	_ "net/http/pprof"
)

var gApp = &App{}

func main() {
	defer logFlush()
	defer gApp.End()

	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	logInit()

	if err := gApp.Init(); err != nil {
		logError(err)
		return
	}
	gApp.Start()
}
