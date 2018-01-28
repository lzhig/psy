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

	"github.com/golang/glog"
)

var gApp = &App{}
var db = &mysqlDB{}
var userManager = &UserManager{}
var loginService = &LoginService{}
var debug func(a ...interface{}) (int, error)

func main() {
	defer logFlush()
	defer gApp.End()

	flag.Parse()

	logInit()

	if err := gApp.Init(); err != nil {
		glog.Errorln(err)
		return
	}
	gApp.Start()
}
