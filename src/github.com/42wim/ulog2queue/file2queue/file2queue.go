package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	_ "github.com/pkg/profile"
	"runtime"
	"time"
)

var cfg *Config
var nrCPU = runtime.GOMAXPROCS(-1)
var log = logrus.New()

func doTask(ctx *Context) {
	log.Info("backend:starting ", ctx.cfg.Backend["es"].Workers, " ES backends")
	for i := 0; i < ctx.cfg.Backend["es"].Workers; i++ {
		go doPrimaryTask(ctx, "es")
	}
	go doRestoreTask(ctx, "disk")
}

func rateLogger(ctx *Context) {
	parsedRatecount := 0
	parsedTotalcount := 0
	pt0 := time.Now()
	pt1 := time.Now()

	for {
		select {
		case <-ctx.parsedRate:
			parsedRatecount++
			parsedTotalcount++
			if time.Since(pt0).Seconds() > 5 {
				log.Info("primary: total: ", parsedTotalcount,
					" rate: ", int(float64(parsedRatecount)/float64(time.Since(pt0).Seconds())), "/s",
					" avg rate: ", int(float64(parsedTotalcount)/float64(time.Since(pt1).Seconds())), "/s",
					" buffer: ", len(ctx.parsedLines))
				pt0 = time.Now()
				parsedRatecount = 0
			}
		}
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func init() {
	var flagDebug bool
	var flagConfig string
	flag.StringVar(&flagConfig, "conf", "ulog2queue.conf", "config file")
	flag.BoolVar(&flagDebug, "debug", false, "enable debug")
	log.Level = logrus.InfoLevel
	flag.Parse()
	if flagDebug {
		log.Println("enabling debug")
		log.Level = logrus.DebugLevel
	}
	cfg = NewConfig(flagConfig)
}

func main() {
	if nrCPU == 1 { // no GOMAXPROCS set
		nrCPU = runtime.NumCPU() / 2
		if nrCPU > 10 {
			nrCPU = 10
		}
		runtime.GOMAXPROCS(nrCPU)
	}
	/*
		context := &Context{make(chan *string, 10000),
			make(chan *[]byte, cfg.General.Buffer),
			make(chan int),
			make(chan int),
			0,
			cfg}
	*/
	context := &Context{make(chan *string, 10000),
		make(chan *[]byte, cfg.General.Buffer),
		make(chan *[]byte, 10000),
		make(chan int),
		make(chan int),
		make(chan string),
		make(chan bool),
		make(chan string),
		0,
		false,
		cfg}

	// do primary and backup tasks
	go doTask(context)

	//show some stats
	rateLogger(context)
}
