package main

import (
	"bufio"
	"os"
	"time"
)

type diskBackend struct {
	f           *os.File
	w           *bufio.Writer
	bulkRequest []*[]byte
	ctx         *Context
}

func NewDiskBackend(ctx *Context, filename string) *diskBackend {
	f, err := os.Create(filename)
	w := bufio.NewWriter(f)
	if err != nil {
		log.Debug(err)
	}
	return &diskBackend{f: f, w: w, ctx: ctx}
}

func (b *diskBackend) BulkAdd(line *[]byte) {
	b.bulkRequest = append(b.bulkRequest, line)
}

func (b *diskBackend) Close() {
	b.f.Close()
}

func (b *diskBackend) Flush() error {
	for _, line := range b.bulkRequest {
		b.w.Write(*line)
		b.w.Write([]byte("\n"))
		b.ctx.backupRate <- 1
	}
	err := b.w.Flush()
	if err != nil {
		return err
	}
	b.bulkRequest = b.bulkRequest[:0]
	return nil
}

func (b *diskBackend) Size() int64 {
	fi, _ := b.f.Stat()
	return fi.Size()
}

func (b *diskBackend) Restore() {
	filename := b.f.Name()
	b.f.Close()
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	rd := bufio.NewReader(f)
	defer f.Close()
	log.Info("disk: restoring ", filename)
	t0 := time.Now()
	i := 0
	for {
		input, err := rd.ReadBytes('\n')
		if err != nil {
			f.Close()
			log.Debug("disk: restored ", i, " removing ", filename)
			os.Remove(filename)
			b.ctx.restoreDone <- true
			return
		}
		input2 := input[:len(input)-1]
		b.ctx.parsedLines <- &input2
		if time.Since(t0).Seconds() > 5 {
			log.Info("disk: restoring ...")
			t0 = time.Now()
		}
		i++
	}
}
