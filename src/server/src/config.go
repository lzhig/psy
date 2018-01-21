/**
 * @author [Bruce]
 * @email [lzhig@outlook.com]
 * @create date 2018-01-19 10:10:59
 * @modify date 2018-01-19 10:10:59
 * @desc [description]
 */

package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
)

type serverConfig struct {
	Addr     string `json:"addr"`
	MaxUsers uint32 `json:"max_users"`
}

// Config type
type Config struct {
	Debug  bool         `json:"debug"`
	CPUNum int          `json:"cpu_num"`
	Server serverConfig `json:"server"`
}

// Load function
func (obj *Config) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, obj)
}
