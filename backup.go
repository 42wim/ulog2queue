package main

import (
	"strconv"
	"time"
)

type Backup interface {
	BulkAdd(line *[]byte)
	Flush() error
	Size() int64
	Close()
	Restore()
}

func doBackupTask(ctx *Context, name string) {
	log.Debug("start backup: ", name)
	backupcount := 0
	i := 0
	bulk := 1000
	var b Backup
	switch name {
	case "disk":
		b = NewDiskBackend(ctx, ctx.cfg.Backend[name].URI[0]+strconv.Itoa(backupcount))
	case "redis":
		b = NewRedisBackend(ctx, ctx.cfg.Backend[name].Queue)
	case "nil":
		b = NewNilBackend(ctx)
	}
	defer b.Close()
	restoring := false
	for {
		select {
		case line := <-ctx.backupLines:
			b.BulkAdd(line)
			if i >= bulk {
				b.Flush()
				i = 0
			}
			i++
		case res := <-ctx.restoreDone:
			restoring = false
			log.Info(name, ": restore done ", res)
		case id := <-ctx.restoreStart:
			if !restoring && b.Size() > 0 {
				restoring = true
				log.Debug(name, ": received start restore from ", id)
				b.Flush()
				go b.Restore()
				backupcount++
				switch name {
				case "disk":
					b = NewDiskBackend(ctx, ctx.cfg.Backend["disk"].URI[0]+strconv.Itoa(backupcount))
				case "redis":
					b = NewRedisBackend(ctx, ctx.cfg.Backend[name].Queue)
				}
			}
		case <-time.After(time.Second * 10):
			b.Flush()
		}
	}
}
