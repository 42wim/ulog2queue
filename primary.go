package main

import (
	"strconv"
	"time"
)

type Primary interface {
	BulkAdd(line *[]byte)
	Flush() error
	Close()
	NumberOfActions() int
	Ping() error
}

func doPrimaryTask(ctx *Context, name string) {
	var stopworking time.Time
	log.Debug("start backup: ", name)
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	id = id[len(id)/2 : len(id)-1]
	bulk := ctx.cfg.Backend[name].Bulk
	working := true
	failed := false

	var p Primary
	switch name {
	case "es":
		p = NewESBackend(ctx, id)
	}
	defer p.Close()
	for {
		select {
		case line := <-ctx.parsedLines:
			if working {
				p.BulkAdd(line)
				if p.NumberOfActions() >= bulk {
					err := p.Flush()
					if err != nil {
						working = false
						failed = true
						stopworking = time.Now()
					} else {
						if failed {
							ctx.restoreStart <- id
							failed = false
						}
						working = true
					}
				}
			} else {
				ctx.backupLines <- line
				failed = true
				// try ES again after 10 seconds
				if time.Since(stopworking).Seconds() > 10 {
					err := p.Ping()
					if err != nil {
						stopworking = time.Now()
						working = false
					} else {
						working = true
					}
				}
			}
		case <-time.After(time.Second * 10):
			if p.NumberOfActions() > 0 && !failed {
				log.Info("ES:", id, ": idle func, flushing data: ", p.NumberOfActions())
				err := p.Flush()
				if err != nil {
					log.Error("ES:", id, ": flush failed, lost events ", err)
				}
			}
			// maybe there's something to restore..
			ctx.restoreStart <- id
		}
	}
}
