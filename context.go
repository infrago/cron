package cron

import (
	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

type (
	Context struct {
		inst *Instance
		infra.Meta

		index int       //下一个索引
		nexts []ctxFunc //方法列表

		// 以下几个字段必须独立
		// 要不然，Invoke的时候，会被修改掉
		// 因为invoke是同一个 *Meta 对象
		Name    string
		Config  *Job
		Value   Map
		Args    Map
		Setting Map

		Body Any
	}
	ctxFunc func(*Context)
)

func (ctx *Context) clear() {
	ctx.index = 0
	ctx.nexts = make([]ctxFunc, 0)
}
func (ctx *Context) next(nexts ...ctxFunc) {
	ctx.nexts = append(ctx.nexts, nexts...)
}

func (ctx *Context) Next() {
	if len(ctx.nexts) > ctx.index {
		next := ctx.nexts[ctx.index]
		ctx.index++
		if next != nil {
			next(ctx)
		} else {
			ctx.Next()
		}
	}
}
