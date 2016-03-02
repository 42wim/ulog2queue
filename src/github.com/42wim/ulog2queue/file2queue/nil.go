package main

type nilBackend struct {
	ctx *Context
}

func NewNilBackend(ctx *Context) *nilBackend {
	return &nilBackend{ctx: ctx}
}

func (b *nilBackend) BulkAdd(line *[]byte) {
	b.ctx.parsedLines <- line
}

func (b *nilBackend) Close() {
}

func (b *nilBackend) Flush() error {
	return nil
}

func (b *nilBackend) Restore() {
}

func (b *nilBackend) Size() int64 {
	return 0
}

func (b *nilBackend) List() []string {
	return nil
}
