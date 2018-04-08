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
}

// Start function
func (obj *VersionService) Start(addr string, filename string) {
	go func() {
		defer base.LogPanic()

		f, err := os.Open(filename)
		if err != nil {
			base.LogError("Failed to open filename:", filename, ". error:", err)
			gApp.Exit()
			return
		}
		defer f.Close()
		data, err := ioutil.ReadAll(bufio.NewReader(f))
		if err != nil {
			base.LogError("Failed to read from filename:", filename, ". error:", err)
			gApp.Exit()
			return
		}

		http.HandleFunc("/get_version", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, string(data))
		})

		http.ListenAndServe(addr, nil)
	}()
}
