package main

import (
	"errors"
	"github.com/pquerna/ffjson/ffjson"
	"gopkg.in/olivere/elastic.v2"
	"strconv"
	"time"
)

type esBackend struct {
	id           string
	c            *elastic.Client
	ctx          *Context
	bulkRequest  *elastic.BulkService
	currentIndex string
	count        uint64
	bulk         int
}

func NewESBackend(ctx *Context, id string) *esBackend {
	client, err := elastic.NewClient(elastic.SetURL(ctx.cfg.Backend["es"].URI[0]), elastic.SetSniff(false))
	if err != nil {
		log.Fatal("ES:", id, ": no node available")
	}
	return &esBackend{ctx: ctx, id: id, c: client, bulkRequest: client.Bulk(),
		count: 1, currentIndex: time.Now().UTC().Format(ctx.cfg.Backend["es"].Index),
		bulk: ctx.cfg.Backend["es"].Bulk}
}

func (b *esBackend) BulkAdd(line *[]byte) {
	if b.count == 0 || b.count%1000 == 0 {
		indextime := b.TimeFromLine(line)
		if b.currentIndex != indextime {
			b.currentIndex = indextime
			log.Info("ES:", b.id, ": index changed to ", b.currentIndex)
		}
	}
	request := elastic.NewBulkIndexRequest().Index(b.currentIndex).Type("json-log").Doc(string(*line)).Id(getFlakeID())
	b.bulkRequest = b.bulkRequest.Add(request)
	b.ctx.parsedRate <- 1
	b.count++
}

func (b *esBackend) TimeFromLine(line *[]byte) string {
	var f nf
	err := ffjson.Unmarshal(*line, &f)
	if err != nil {
		return time.Now().UTC().Format(b.ctx.cfg.Backend["es"].Index)
	}
	t, _ := time.Parse(time.RFC3339Nano, f.Ltimestamp)
	return t.Format(b.ctx.cfg.Backend["es"].Index)
}

func (b *esBackend) Close() {
	return
}

func (b *esBackend) Flush() error {
	log.Debug("ES:", b.id, ": sending bulk ", b.bulk, ":", b.bulkRequest.NumberOfActions())
	_, err := b.bulkRequest.Do()
	if err != nil {
		log.Error("ES:", b.id, ": bulkresponse error ", err)
		return err
	}
	return nil
}

func (b *esBackend) NumberOfActions() int {
	return b.bulkRequest.NumberOfActions()
}

func (b *esBackend) Ping() error {
	_, status, _ := b.c.Ping().URL(b.ctx.cfg.Backend["es"].URI[0]).HttpHeadOnly(true).Do()
	if status != 200 {
		log.Debug("ES:", b.id, ": notworking, sending to backup")
		return errors.New("status: " + strconv.Itoa(status))
	}
	return nil
}
