package main

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

var (
	redisPool *redis.Pool
)

type redisBackend struct {
	bulkRequest []*[]byte
	ctx         *Context
	queue       string
}

func NewPool(ctx *Context) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     20,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ctx.cfg.Backend["redis"].URI[0])
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func NewRedisBackend(ctx *Context, queue string) *redisBackend {
	return &redisBackend{ctx: ctx, queue: queue}
}

func (b *redisBackend) BulkAdd(line *[]byte) {
	b.bulkRequest = append(b.bulkRequest, line)
}

func (b *redisBackend) Close() {
	return
}

func (b *redisBackend) Flush() error {
	c := redisPool.Get()
	defer c.Close()
	ctx := b.ctx
	bulkRequest := b.bulkRequest
	c.Send("MULTI")
	for _, line := range bulkRequest {
		c.Send("LPUSH", b.queue, *line)
		ctx.backupRate <- 1
	}
	_, err := c.Do("EXEC")
	if err != nil {
		// just fail here, systemd should restart us
		log.Fatal("redis: could not connect", err)
		return err
	}
	b.bulkRequest = b.bulkRequest[:0]
	return nil
}

func (b *redisBackend) NumberOfActions() int {
	return len(b.bulkRequest)
}

func (b *redisBackend) Ping() error {
	c := redisPool.Get()
	defer c.Close()
	_, err := c.Do("PING")
	return err
}

func (b *redisBackend) Restore() {
	c := redisPool.Get()
	ctx := b.ctx
	defer c.Close()
	log.Info("redis: restoring ", b.queue)
	t0 := time.Now()

	for {
		if time.Since(t0).Seconds() > 5 {
			log.Info("redis: restoring ...")
			t0 = time.Now()
		}
		reply, err := redis.Bytes(c.Do("RPOP", b.queue))
		if err != nil {
			ctx.restoreDone <- true
			return
		}
		for {
			if ctx.buffering {
				log.Debug("we're buffering, wait restoring..")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
		ctx.parsedLines <- &reply
	}
}

func (b *redisBackend) Size() int64 {
	c := redisPool.Get()
	defer c.Close()
	status, _ := redis.Int64(c.Do("LLEN", b.queue))
	return status
}
