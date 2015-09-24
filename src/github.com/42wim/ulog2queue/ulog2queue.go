package main

import (
	"flag"
	"fmt"
	"github.com/42wim/tail"
	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/golang-lru"
	"github.com/oschwald/geoip2-golang"
	_ "github.com/pkg/profile"
	"github.com/pquerna/ffjson/ffjson"
	"net"
	"os"
	"runtime"
	"time"
)

var flagPrimary, flagQueue, flagBackup, flagTailFile string
var geoipDB *geoip2.Reader
var geoipCache *lru.Cache
var cfg *Config
var nrCPU = runtime.GOMAXPROCS(-1)
var log = logrus.New()

const nfLayout = "2006-01-02T15:04:05.999999999"

var myLocation *time.Location

func parseLine(line *[]byte) {
	var f nf
	var t time.Time
	var realRegionName, regionName string
	var record *geoip2.City

	err := ffjson.Unmarshal(*line, &f)
	if err != nil {
		*line = []byte("")
		log.Error(err, "couldn't unmarshal ", *line)
		return
	}

	// use LRU cache
	if val, ok := geoipCache.Get(*f.Srcip); ok {
		record = val.(*geoip2.City)
	} else {
		ip := net.ParseIP(*f.Srcip)
		record, _ = geoipDB.City(ip)
		geoipCache.Add(*f.Srcip, record)
	}

	// add @timestamp with zulu (ISO8601 time)
	t, _ = time.ParseInLocation(nfLayout, *f.Timestamp, myLocation)
	f.Ltimestamp = t.UTC().Format(time.RFC3339Nano)

	if record.Location.Longitude != 0 && record.Location.Latitude != 0 {
		mylen := len(record.Subdivisions)
		if mylen > 0 {
			mylen--
			realRegionName = record.Subdivisions[mylen].Names["en"]
			regionName = record.Subdivisions[mylen].IsoCode
		}
		f.GeoIP.Longitude = &record.Location.Longitude
		f.GeoIP.Latitude = &record.Location.Latitude
		f.GeoIP.CountryName = record.Country.Names["en"]
		f.GeoIP.Timezone = &record.Location.TimeZone
		f.GeoIP.ContinentCode = &record.Continent.Code
		f.GeoIP.CityName = record.City.Names["en"]
		f.GeoIP.CountryCode2 = &record.Country.IsoCode
		f.GeoIP.RealRegionName = &realRegionName
		f.GeoIP.RegionName = &regionName
		f.GeoIP.IP = f.Srcip
		f.GeoIP.Location = &esGeoIPLocation{f.GeoIP.Longitude, f.GeoIP.Latitude}
		f.GeoIP.Coordinates = f.GeoIP.Location
	}

	*line, _ = ffjson.Marshal(f)
}

func parseLineWorker(ctx *Context) {
	bt0 := time.Now()
	show := true
	for {
		select {
		case line := <-ctx.lines:
			myline := []byte(*line)
			parseLine(&myline)
			// primary buffer full, send to backup
			if len(ctx.parsedLines) > ctx.cfg.General.Buffer*90/100 {
				ctx.buffering = true
				if show {
					log.Info("primary isn't fast enough, buffer full, buffering to backup")
					show = false
				}
				// only log message every 5 seconds
				if time.Since(bt0).Seconds() > 5 {
					show = true
					bt0 = time.Now()
				}
				ctx.backupLines <- &myline
			} else {
				ctx.buffering = false
				ctx.parsedLines <- &myline
			}
		case <-time.After(time.Second * 3):
			ctx.buffering = false
		}
	}
}

func doTask(ctx *Context) {
	switch flagPrimary {
	case "redis", "rabbit":
		go doPrimaryTask(ctx, flagPrimary)
	case "es": // es is pretty slow, start multiple
		log.Info("backend:starting ", ctx.cfg.Backend["es"].Workers, " ES backends")
		for i := 0; i < ctx.cfg.Backend["es"].Workers; i++ {
			go doPrimaryTask(ctx, "es")
		}
	}
	if flagBackup != "" {
		go doBackupTask(ctx, flagBackup)
	}
}

func rateLogger(ctx *Context) {
	parsedRatecount := 0
	backupRatecount := 0
	parsedTotalcount := 0
	backupTotalcount := 0
	pt0 := time.Now()
	pt1 := time.Now()

	bt0 := time.Now()
	bt1 := time.Now()
	ticker := time.NewTicker(5 * time.Second)
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
		case <-ctx.backupRate:
			backupRatecount++
			backupTotalcount++
			if time.Since(bt0).Seconds() > 5 {
				log.Info("backup: total: ", backupTotalcount,
					" rate: ", int(float64(backupRatecount)/float64(time.Since(bt0).Seconds())), "/s",
					" avg rate: ", int(float64(backupTotalcount)/float64(time.Since(bt1).Seconds())), "/s",
					" buffer: ", len(ctx.parsedLines))
				bt0 = time.Now()
				backupRatecount = 0
			}
		case <-ticker.C:
			//log.Debug(backupRatecount, ":", ctx.backupRateInt)
			if backupRatecount == ctx.backupRateInt {
				//log.Debug("ratelogger: sending restoreStart")
				ctx.restoreStart <- "ratelogger"
			} else {
				ctx.backupRateInt = backupRatecount
			}
		}
	}
}

func tailUlog(ctx *Context) {
	logfile := flagTailFile
	t, err := tail.TailFile(logfile, tail.Config{Poll: true, Follow: true, ReOpen: true})
	if err != nil {
		log.Error(err)
	}

	// create the workers. line goes in, parsed line goes out
	for i := 0; i < nrCPU/2; i++ {
		go parseLineWorker(ctx)
	}
	// do primary and backup tasks
	go doTask(ctx)

	//show some stats
	go rateLogger(ctx)

	for line := range t.Lines {
		ctx.lines <- &line.Text
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
	flag.StringVar(&flagPrimary, "primary", "", "name of primary backend (es/redis/rabbit)")
	flag.StringVar(&flagBackup, "backup", "", "name of backup backend (disk/redis/rabbit)")
	flag.StringVar(&flagConfig, "conf", "ulog2queue.cfg", "config file")
	flag.StringVar(&flagTailFile, "tail", "", "file to tail")
	flag.BoolVar(&flagDebug, "debug", false, "enable debug")
	hostname, _ := os.Hostname()
	flag.StringVar(&flagQueue, "queue", hostname, "name of queue")
	log.Level = logrus.InfoLevel
	flag.Parse()
	if flagDebug {
		log.Println("enabling debug")
		log.Level = logrus.DebugLevel
	}
	cfg = NewConfig(flagConfig)
	if flagPrimary == "" {
		flagPrimary = cfg.General.Primary
	}
	if flagBackup == "" {
		flagBackup = cfg.General.Backup
	}
	if flagTailFile == "" {
		flagTailFile = cfg.General.TailFile
	}
	myLocation, _ = time.LoadLocation("Local")
	geoipCache, _ = lru.New(10000)
}

func main() {
	var err error
	if nrCPU == 1 { // no GOMAXPROCS set
		nrCPU = runtime.NumCPU() / 2
		if nrCPU > 10 {
			nrCPU = 10
		}
		runtime.GOMAXPROCS(nrCPU)
	}
	context := &Context{make(chan *string, 10000),
		make(chan *[]byte, 10000),
		make(chan *[]byte, cfg.General.Buffer),
		make(chan int),
		make(chan int),
		make(chan string),
		make(chan bool),
		make(chan string),
		0,
		false,
		cfg}
	geoipDB, err = geoip2.Open(context.cfg.General.Geoip2db)
	failOnError(err, "can't open geoip db")
	redisPool = NewPool(context)
	tailUlog(context)
}
