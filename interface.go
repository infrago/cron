package cron

import (
	"fmt"
	"log"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
	"github.com/robfig/cron/v3"
)

func (this *Module) Register(name string, value Any) {
	switch config := value.(type) {
	case Job:
		this.Job(name, config)
	case Filter:
		this.Filter(name, config)
	}
}

func (this *Module) Configure(global Map) {
	var config Map
	if vvv, ok := global["corn"].(Map); ok {
		config = vvv
	}
	if config == nil {
		return
	}

	if setting, ok := config["setting"].(Map); ok {
		this.config.Setting = setting
	}
}
func (this *Module) Initialize() {
	if this.initialized {
		return
	}

	//时间记录
	for key, job := range this.jobs {
		times := make([]string, 0)
		if job.Time != "" {
			times = append(times, job.Time)
		}
		if job.Times != nil || len(job.Times) > 0 {
			times = append(times, job.Times...)
		}
		this.jobTimes[key] = times
	}

	//拦截器
	this.filterActions = make([]ctxFunc, 0)
	for _, filter := range this.filters {
		if filter.Action != nil {
			this.filterActions = append(this.filterActions, filter.Action)
		}
	}

	this.initialized = true
}
func (this *Module) Connect() {
	if this.connected {
		return
	}

	this.cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	// this.cronEntries = make(map[string][]string, 0)

	inst := &Instance{
		// this,
	}

	for key, times := range this.jobTimes {
		name := key

		// ids := make([]string, 0)
		for _, crontab := range times {
			// timeName := fmt.Sprintf("%s.%v", key, i)
			_, err := this.cron.AddFunc(crontab, func() {
				inst.Serve(name)
			})

			if err != nil {
				panic("[cron]注册失败：" + err.Error())
			}

			// ids = append(ids, id)
		}

		// this.cronEntries[name] = ids
	}

	this.instance = inst

	this.connected = true
}
func (this *Module) Launch() {
	if this.launched {
		return
	}

	this.cron.Start()

	log.Println(fmt.Sprintf("%s CRON is running with %d jobs.", infra.INFRAGO, len(this.jobs)))

	this.launched = true
}
func (this *Module) Terminate() {
	if this.cron != nil {
		this.cron.Stop()
	}

	this.launched = false
	this.connected = false
	this.initialized = false
}
