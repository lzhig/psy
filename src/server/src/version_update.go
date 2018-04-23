package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/lzhig/rapidgo/base"
)

// VersionService type
type VersionService struct {
	updateChan chan struct{}
}

// UpdateVersionInfo type
func (obj *VersionService) UpdateVersionInfo() {
	obj.updateChan <- struct{}{}
}

// Start function
func (obj *VersionService) Start(addr string, filename string) {
	obj.updateChan = make(chan struct{}, 1)
	go func() {
		defer base.LogPanic()

		data, err := obj.loadVersionInfo(filename)
		if err != nil {
			gApp.Exit()
			return
		}

		http.HandleFunc("/get_version", func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-obj.updateChan:
				data, err = obj.loadVersionInfo(filename)
			default:
			}
			io.WriteString(w, data)
		})

		http.ListenAndServe(addr, nil)
	}()
}

func (obj *VersionService) loadVersionInfo(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		base.LogError("Failed to open filename:", filename, ". error:", err)
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(bufio.NewReader(f))
	if err != nil {
		base.LogError("Failed to read from filename:", filename, ". error:", err)
		return "", err
	}
	return string(data), err
}
