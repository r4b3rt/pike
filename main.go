package main

import (
	"flag"
	"io/ioutil"
	"log"
	"runtime"

	"./cache"
	"./director"
	"./server"
	"./vars"
	"gopkg.in/yaml.v2"
)

var (
	config string
)

func main() {
	flag.StringVar(&config, "c", "/etc/pike/config.yml", "the config file")
	flag.Parse()
	buf, err := ioutil.ReadFile(config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	conf := &server.PikeConfig{}
	err = yaml.Unmarshal(buf, conf)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if conf.Cpus > 0 {
		runtime.GOMAXPROCS(conf.Cpus)
	}

	db, err := cache.InitDB(conf.DB)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer db.Close()
	directorList := director.GetDirectors(conf.Directors)
	for _, d := range directorList {
		name := d.Name
		cache.InitBucket([]byte(name))
	}
	// 用于保存配置数据
	cache.InitBucket(vars.ConfigBucket)
	err = server.Start(conf, directorList)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
