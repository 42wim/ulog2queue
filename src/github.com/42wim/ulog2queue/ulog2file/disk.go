package main

import (
	"bufio"
	"compress/gzip"
	"os"
	"time"
)

type diskBackend struct {
	f           *os.File
	fgz         *os.File
	w           *bufio.Writer
	gzw         *gzip.Writer
	bulkRequest []*[]byte
	ctx         *Context
	unc         string
}

func NewDiskBackend(ctx *Context, filename string, directory string) *diskBackend {
	fgz, err := os.Create(directory + "/" + filename)
	f, err := os.Create(directory + "/unc/" + filename)
	w := bufio.NewWriter(f)
	gzw, _ := gzip.NewWriterLevel(fgz, gzip.BestCompression)
	if err != nil {
		log.Debug(err)
	}
	return &diskBackend{fgz: fgz, f: f, gzw: gzw, w: w, ctx: ctx, unc: directory + "/unc/" + filename}
}

func (b *diskBackend) BulkAdd(line *[]byte) {
	b.bulkRequest = append(b.bulkRequest, line)
}

func (b *diskBackend) Close() {
	b.gzw.Close()
	b.fgz.Close()
	b.f.Close()
	go func() {
		time.Sleep(time.Second * 30)
		os.Remove(b.unc)
	}()
}

func (b *diskBackend) Flush() error {
	for _, line := range b.bulkRequest {
		b.gzw.Write(*line)
		b.w.Write(*line)
		b.gzw.Write([]byte("\n"))
		b.w.Write([]byte("\n"))
		b.ctx.backupRate <- 1
	}
	err := b.gzw.Flush()
	if err != nil {
		return err
	}
	err = b.w.Flush()
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
}
