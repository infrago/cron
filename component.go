package cron

import (
	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

type (
	Job struct {
		Name    string
		Text    string
		Time    string
		Times   []string
		Setting Map  `json:"-"`
		Coding  bool `json:"-"`

		Action  ctxFunc   `json:"-"`
		Actions []ctxFunc `json:"-"`
	}

	// Filter 拦截器
	Filter struct {
		Name   string  `json:"name"`
		Text   string  `json:"text"`
		Action ctxFunc `json:"-"`
	}
)

func (this *Module) Job(name string, config Job) {
	if infra.Override() {
		this.jobs[name] = config
	} else {
		if _, ok := this.jobs[name]; ok == false {
			this.jobs[name] = config
		}
	}
}

// Filter 注册 拦截器
func (this *Module) Filter(name string, config Filter) {
	if infra.Override() {
		this.filters[name] = config
	} else {
		if _, ok := this.filters[name]; ok == false {
			this.filters[name] = config
		}
	}
}
