/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-18 03:43:40
 * @modify date 2018-01-18 03:43:40
 * @desc [description]
 */
package main

import "github.com/golang/glog"

var gApp = &App{}

func main() {
	if err := gApp.Init(); err != nil {
		glog.Errorln(err)
		return
	}
	gApp.Start()
}
