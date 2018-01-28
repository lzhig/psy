package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
)

func logInit() {
	// 设置log目录
	logDir := "./log"
	flag.CommandLine.Set("log_dir", logDir)
	os.Mkdir(logDir, os.ModePerm)
}

func logInfo(args ...interface{}) {
	glog.Infoln(args...)
}

func logWarn(args ...interface{}) {
	glog.Warningln(args...)
}

func logError(args ...interface{}) {
	glog.Errorln(args...)
}

func logFlush() {
	glog.Flush()
}
