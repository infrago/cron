package cron

import (
	"time"

	"github.com/infrago/infra"
	"github.com/infrago/mutex"
)

func (this *Instance) Serve(name string) {
	//是否考虑内置？要不然还依赖模块
	if mutex.Locked("cron", name, time.Now().Unix(), time.Second) {
		return //加锁，防止多节点并发多次调用
	}

	config, ok := module.jobs[name]
	if ok == false {
		return
	}

	ctx := &Context{inst: this}
	ctx.Name = name
	ctx.Config = &config
	ctx.Setting = config.Setting

	// 解析元数据
	metadata := infra.Metadata{}
	ctx.Metadata(metadata)

	this.execute(ctx)

	infra.CloseMeta(&ctx.Meta)
}

func (this *Instance) execute(ctx *Context) {
	ctx.clear()

	//拦截器
	ctx.next(module.filterActions...)
	if ctx.Config.Actions != nil || len(ctx.Config.Actions) > 0 {
		ctx.next(ctx.Config.Actions...)
	}
	if ctx.Config.Action != nil {
		ctx.next(ctx.Config.Action)
	}

	//开始执行
	ctx.Next()
}
