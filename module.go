package cron

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/base"
	robcron "github.com/robfig/cron/v3"
)

func init() {
	bamgoo.Mount(module)
}

var module = &Module{
	config: Config{
		Driver:  bamgoo.DEFAULT,
		Tick:    time.Second,
		Sync:    time.Second * 5,
		LockTTL: time.Second * 30,
	},
	drivers:   make(map[string]Driver),
	jobs:      make(map[string]Job),
	schedules: make(map[string][]*scheduleItem),
}

type (
	Module struct {
		mutex sync.RWMutex

		config Config

		drivers map[string]Driver
		jobs    map[string]Job

		instance *Instance
		connect  Connection

		parser    robcron.Parser
		schedules map[string][]*scheduleItem

		running bool
		stopCh  chan struct{}
		wg      sync.WaitGroup
	}

	Config struct {
		Driver  string
		Tick    time.Duration
		Sync    time.Duration
		LockTTL time.Duration
		Setting Map
	}

	Instance struct {
		Config  Config
		Setting Map
	}

	Job struct {
		Name      string   `json:"name"`
		Desc      string   `json:"desc"`
		Target    string   `json:"target"`
		Schedule  string   `json:"schedule"`
		Schedules []string `json:"schedules"`
		Disabled  bool     `json:"disabled"`
		Payload   Map      `json:"payload"`
		Setting   Map      `json:"setting"`
	}

	Jobs map[string]Job

	Log struct {
		Job       string    `json:"job"`
		Schedule  string    `json:"schedule"`
		Target    string    `json:"target"`
		Payload   Map       `json:"payload"`
		Triggered time.Time `json:"triggered"`
		Started   time.Time `json:"started"`
		Ended     time.Time `json:"ended"`
		Success   bool      `json:"success"`
		State     string    `json:"state"`
		Error     string    `json:"error"`
	}

	scheduleItem struct {
		jobName  string
		spec     string
		schedule robcron.Schedule
		next     time.Time
	}
)

func (m *Module) Register(name string, value Any) {
	switch v := value.(type) {
	case Driver:
		m.RegisterDriver(name, v)
	case Config:
		m.RegisterConfig(v)
	case Job:
		m.RegisterJob(name, v)
	case Jobs:
		m.RegisterJobs(v)
	}
}

func (m *Module) RegisterDriver(name string, driver Driver) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		name = bamgoo.DEFAULT
	}
	if driver == nil {
		panic("invalid cron driver: " + name)
	}
	if bamgoo.Override() {
		m.drivers[name] = driver
	} else if _, ok := m.drivers[name]; !ok {
		m.drivers[name] = driver
	}
}

func (m *Module) RegisterConfig(config Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if config.Driver != "" {
		m.config.Driver = config.Driver
	}
	if config.Tick > 0 {
		m.config.Tick = config.Tick
	}
	if config.Sync > 0 {
		m.config.Sync = config.Sync
	}
	if config.LockTTL > 0 {
		m.config.LockTTL = config.LockTTL
	}
	if config.Setting != nil {
		m.config.Setting = cloneMap(config.Setting)
	}
}

func (m *Module) RegisterJob(name string, job Job) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		return
	}
	job.Name = name
	m.jobs[name] = cloneJob(job)
}

func (m *Module) RegisterJobs(jobs Jobs) {
	for name, job := range jobs {
		m.RegisterJob(name, job)
	}
}

func (m *Module) Config(global Map) {
	cfgAny, ok := global["cron"]
	if !ok {
		return
	}
	cfgMap, ok := cfgAny.(Map)
	if !ok || cfgMap == nil {
		return
	}

	cfg := Config{}
	if v, ok := cfgMap["driver"].(string); ok {
		cfg.Driver = v
	}
	if v, ok := cfgMap["tick"]; ok {
		if d := parseDuration(v); d > 0 {
			cfg.Tick = d
		}
	}
	if v, ok := cfgMap["sync"]; ok {
		if d := parseDuration(v); d > 0 {
			cfg.Sync = d
		}
	}
	if v, ok := cfgMap["reload"]; ok {
		if d := parseDuration(v); d > 0 {
			cfg.Sync = d
		}
	}
	if v, ok := cfgMap["lockttl"]; ok {
		if d := parseDuration(v); d > 0 {
			cfg.LockTTL = d
		}
	}
	if v, ok := cfgMap["setting"].(Map); ok {
		cfg.Setting = v
	}
	m.RegisterConfig(cfg)
}

func (m *Module) Setup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.config.Driver == "" {
		m.config.Driver = bamgoo.DEFAULT
	}
	if m.config.Tick <= 0 {
		m.config.Tick = time.Second
	}
	if m.config.Sync <= 0 {
		m.config.Sync = time.Second * 5
	}
	if m.config.LockTTL <= 0 {
		m.config.LockTTL = time.Second * 30
	}
	m.parser = robcron.NewParser(
		robcron.SecondOptional | robcron.Minute | robcron.Hour | robcron.Dom | robcron.Month | robcron.Dow | robcron.Descriptor,
	)
}

func (m *Module) Open() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.connect != nil {
		return
	}

	driver, ok := m.drivers[m.config.Driver]
	if !ok || driver == nil {
		panic("missing cron driver: " + m.config.Driver)
	}

	inst := &Instance{
		Config:  m.config,
		Setting: cloneMap(m.config.Setting),
	}
	conn, err := driver.Connection(inst)
	if err != nil {
		panic("connect cron failed: " + err.Error())
	}
	if err := conn.Open(); err != nil {
		panic("open cron failed: " + err.Error())
	}
	m.instance = inst
	m.connect = conn

	stored, err := conn.List()
	if err == nil {
		for name, job := range stored {
			if _, exists := m.jobs[name]; !exists || bamgoo.Override() {
				m.jobs[name] = cloneJob(job)
			}
		}
	}
	for name, job := range m.jobs {
		_ = conn.Add(name, job)
	}
	m.rebuildSchedulesLocked()
}

func (m *Module) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})

	m.wg.Add(1)
	go m.loop()
	fmt.Printf("bamgoo cron module is running with %d jobs.\n", len(m.jobs))
}

func (m *Module) Stop() {
	m.mutex.Lock()
	if !m.running {
		m.mutex.Unlock()
		return
	}
	close(m.stopCh)
	m.running = false
	m.mutex.Unlock()

	m.wg.Wait()
}

func (m *Module) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.connect != nil {
		_ = m.connect.Close()
		m.connect = nil
	}
	m.instance = nil
	m.schedules = make(map[string][]*scheduleItem)
}

func (m *Module) Upsert(name string, job Job) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		return fmt.Errorf("job name is required")
	}
	job.Name = name
	m.jobs[name] = cloneJob(job)
	if m.connect != nil {
		if err := m.connect.Add(name, job); err != nil {
			return err
		}
	}
	m.rebuildSchedulesLocked()
	return nil
}

func (m *Module) Delete(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.jobs, name)
	delete(m.schedules, name)
	if m.connect != nil {
		return m.connect.Remove(name)
	}
	return nil
}

func (m *Module) Enable(name string) error {
	return m.setDisabled(name, false)
}

func (m *Module) Disable(name string) error {
	return m.setDisabled(name, true)
}

func (m *Module) setDisabled(name string, disabled bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	job, ok := m.jobs[name]
	if !ok {
		return fmt.Errorf("job %s not found", name)
	}
	if job.Disabled == disabled {
		return nil
	}

	job.Disabled = disabled
	m.jobs[name] = cloneJob(job)

	if m.connect == nil {
		return nil
	}
	if disabled {
		return m.connect.Disable(name)
	}
	return m.connect.Enable(name)
}

func (m *Module) loop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Tick)
	syncTicker := time.NewTicker(m.config.Sync)
	defer ticker.Stop()
	defer syncTicker.Stop()

	for {
		select {
		case <-ticker.C:
			m.dispatchDue(time.Now())
		case <-syncTicker.C:
			m.syncFromStore()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Module) dispatchDue(now time.Time) {
	m.mutex.Lock()
	type due struct {
		spec    string
		dueTime time.Time
		job     Job
	}
	dueList := make([]due, 0)

	for jobName, items := range m.schedules {
		job, ok := m.jobs[jobName]
		if !ok {
			continue
		}
		for _, item := range items {
			for !item.next.After(now) {
				d := due{
					spec:    item.spec,
					dueTime: item.next,
					job:     cloneJob(job),
				}
				dueList = append(dueList, d)
				item.next = item.schedule.Next(item.next)
			}
		}
	}
	m.mutex.Unlock()

	for _, d := range dueList {
		m.execute(d.job, d.spec, d.dueTime)
	}
}

func (m *Module) execute(job Job, spec string, dueTime time.Time) {
	m.mutex.RLock()
	conn := m.connect
	lockTTL := m.config.LockTTL
	m.mutex.RUnlock()

	if job.Disabled || job.Target == "" || conn == nil {
		return
	}

	lockKey := fmt.Sprintf("cron:%s:%s:%d", job.Name, spec, dueTime.Unix())
	locked, err := conn.Lock(lockKey, lockTTL)
	if err != nil || !locked {
		return
	}

	go func() {
		started := time.Now()
		meta := bamgoo.NewMeta()
		payload := cloneMap(job.Payload)
		if payload == nil {
			payload = Map{}
		}
		_ = meta.Invoke(job.Target, payload)
		res := meta.Result()
		ended := time.Now()

		log := Log{
			Job:       job.Name,
			Schedule:  spec,
			Target:    job.Target,
			Payload:   cloneMap(payload),
			Triggered: dueTime,
			Started:   started,
			Ended:     ended,
			Success:   true,
		}
		if res != nil {
			log.State = res.State()
			if res.Fail() {
				log.Success = false
				log.Error = res.Error()
			}
		}
		_ = conn.AppendLog(log)
	}()
}

func (m *Module) syncFromStore() {
	m.mutex.RLock()
	conn := m.connect
	m.mutex.RUnlock()
	if conn == nil {
		return
	}

	stored, err := conn.List()
	if err != nil {
		return
	}

	latest := make(map[string]Job, len(stored))
	for name, job := range stored {
		job.Name = name
		latest[name] = cloneJob(job)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if jobsEqual(m.jobs, latest) {
		return
	}
	m.jobs = latest
	m.rebuildSchedulesLocked()
}

func (m *Module) rebuildSchedulesLocked() {
	m.schedules = make(map[string][]*scheduleItem, len(m.jobs))
	now := time.Now()

	for name, job := range m.jobs {
		specs := collectSpecs(job)
		if len(specs) == 0 {
			continue
		}
		items := make([]*scheduleItem, 0, len(specs))
		for _, spec := range specs {
			schedule, err := m.parser.Parse(spec)
			if err != nil {
				continue
			}
			items = append(items, &scheduleItem{
				jobName:  name,
				spec:     spec,
				schedule: schedule,
				next:     schedule.Next(now),
			})
		}
		if len(items) > 0 {
			m.schedules[name] = items
		}
	}
}

func collectSpecs(job Job) []string {
	specs := make([]string, 0, 1+len(job.Schedules))
	if job.Schedule != "" {
		specs = append(specs, job.Schedule)
	}
	specs = append(specs, job.Schedules...)
	uniq := make([]string, 0, len(specs))
	seen := map[string]struct{}{}
	for _, spec := range specs {
		if spec == "" {
			continue
		}
		if _, ok := seen[spec]; ok {
			continue
		}
		seen[spec] = struct{}{}
		uniq = append(uniq, spec)
	}
	return uniq
}

func parseDuration(v Any) time.Duration {
	switch vv := v.(type) {
	case time.Duration:
		return vv
	case int:
		return time.Second * time.Duration(vv)
	case int64:
		return time.Second * time.Duration(vv)
	case float64:
		return time.Second * time.Duration(vv)
	case string:
		d, err := time.ParseDuration(vv)
		if err == nil {
			return d
		}
	}
	return 0
}

func jobsEqual(a, b map[string]Job) bool {
	if len(a) != len(b) {
		return false
	}
	for name, av := range a {
		bv, ok := b[name]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}

func (m *Module) GetLogs(jobName string, offset, limit int) (int64, []Log) {
	m.mutex.RLock()
	conn := m.connect
	m.mutex.RUnlock()

	if conn == nil {
		return 0, []Log{}
	}
	total, logs, err := conn.History(jobName, offset, limit)
	if err != nil {
		return 0, []Log{}
	}
	return total, logs
}
