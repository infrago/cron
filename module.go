package cron

import (
	"sync"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
	"github.com/robfig/cron/v3"
)

func init() {
	infra.Mount(module)
}

var (
	module = &Module{
		config: Config{},

		jobs:    make(map[string]Job, 0),
		filters: make(map[string]Filter, 0),

		jobTimes:      make(map[string][]string, 0),
		filterActions: make([]ctxFunc, 0),

		cronEntries: make(map[string][]string, 0),
	}
)

type (
	Module struct {
		mutex sync.Mutex

		connected, initialized, launched bool

		config Config

		jobs    map[string]Job
		filters map[string]Filter

		jobTimes      map[string][]string
		filterActions []ctxFunc

		instance    *Instance
		cron        *cron.Cron
		cronEntries map[string][]string
	}

	Config struct {
		Setting Map
	}
	Instance struct {
		// module *Module
	}
)
