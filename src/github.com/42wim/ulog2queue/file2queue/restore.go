package main

type Restore interface {
	Restore()
	RestoreTail(string)
	List() []string
}

func doRestoreTask(ctx *Context, name string) {
	log.Debug("start restore: ", name)
	var b Restore
	for {
		b = NewDiskBackend(ctx, ctx.cfg.Backend[name].URI[0])
		b.Restore()
	}
}
