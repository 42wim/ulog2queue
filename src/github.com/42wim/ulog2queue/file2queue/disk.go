package main

import (
	"bufio"
	"github.com/42wim/tail"
	gzip "github.com/klauspost/pgzip"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

type diskBackend struct {
	filename  string
	directory string
	ctx       *Context
	tail      bool
}

func NewDiskBackend(ctx *Context, directory string) *diskBackend {
	b := &diskBackend{ctx: ctx, directory: directory}
	b.filename = b.List()[0]
	current, _ := ioutil.ReadFile(directory + "/meta/current")
	if b.filename == string(current) {
		b.tail = true
	}
	return b
}

func (b *diskBackend) Restore() {
	if b.tail {
		log.Info("restoring with tail ", b.directory+"/unc/"+b.filename)
		b.RestoreTail(b.directory + "/unc/" + b.filename)
		return
	}
	log.Info("restoring gzipped file ", b.directory+"/"+b.filename)
	f, err := os.Open(b.directory + "/" + b.filename)
	if err != nil {
		return
	}
	gzrd, _ := gzip.NewReader(f)
	rd := bufio.NewReader(gzrd)
	defer gzrd.Close()
	defer f.Close()
	t0 := time.Now()
	i := 1
	for {
		input, err := rd.ReadBytes('\n')
		if err != nil {
			f.Close()
			log.Info("disk: restored ", i, " removing ", b.filename)
			os.Remove(b.directory + "/" + b.filename)
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

type timeSlice []os.FileInfo

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].ModTime().Before(p[j].ModTime())
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (b *diskBackend) List() []string {
	var files []string
	dir, _ := os.Open(b.directory)
	listing, _ := dir.Readdir(-1)
	sort.Sort(timeSlice(listing))
	for _, f := range listing {
		if f.IsDir() == false {
			files = append(files, f.Name())
		}
	}
	return files
}

func (b *diskBackend) RestoreTail(filename string) {
	t, err := tail.TailFile(filename, tail.Config{Poll: true, Follow: true, ReOpen: false, Pipe: false})
	if err != nil {
		log.Error(err)
	}
	i := 0
	for line := range t.Lines {
		l := []byte(line.Text)
		b.ctx.parsedLines <- &l
		i++
		if i == 500000 {
			log.Debug(string(l))
			log.Info("tail read 500000 lines, breaking")
			break
		}
	}
	t.Stop()
	log.Info("removing ", filename)
	os.Remove(b.directory + "/" + b.filename)
	os.Remove(filename)
	time.Sleep(time.Second * 2)
}
