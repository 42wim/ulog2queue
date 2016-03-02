package main

import (
	"io/ioutil"
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
	filename := ctx.cfg.Backend[name].Queue
	directory := ctx.cfg.Backend[name].URI[0]
	//directory := "/var/log/ulog2file"
	result, _ := ioutil.ReadFile(directory + "/meta/status")
	backupcount, err := strconv.Atoi(string(result))
	log.Debug(backupcount, string(result), err)
	backupcount++
	i := 0
	j := 0
	bulk := 1000
	var b Backup
	b = NewDiskBackend(ctx, filename+strconv.Itoa(backupcount), directory)
	ioutil.WriteFile(directory+"/meta/current", []byte(filename+strconv.Itoa(backupcount)), 0666)
	defer b.Close()
	for {
		select {
		case line := <-ctx.backupLines:
			b.BulkAdd(line)
			if i >= bulk {
				b.Flush()
				i = 0
				if j%500000 == 0 {
					log.Info("size > 500000 entries - creating: ", filename+strconv.Itoa(backupcount))
					backupcount++
					b.Close()
					b = NewDiskBackend(ctx, filename+strconv.Itoa(backupcount), directory)
					ioutil.WriteFile(directory+"/meta/current", []byte(filename+strconv.Itoa(backupcount)), 0666)
					ioutil.WriteFile(directory+"/meta/status", []byte(strconv.Itoa(backupcount)), 0666)
				}
			}
			i++
			j++
		case <-time.After(time.Second * 10):
			b.Flush()
		}
	}
}
