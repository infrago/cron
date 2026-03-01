# cron

`cron` 是 infrago 的模块包。

## 安装

```bash
go get github.com/infrago/cron@latest
```

## 最小接入

```go
package main

import (
    _ "github.com/infrago/cron"
    "github.com/infrago/infra"
)

func main() {
    infra.Run()
}
```

## 配置示例

```toml
[cron]
driver = "default"
```

## 公开 API（摘自源码）

- `func (Job) RegistryComponent() string`
- `func (Jobs) RegistryComponent() string`
- `func (d *defaultDriver) Connection(_ *Instance) (Connection, error)`
- `func (c *defaultConnect) Open() error  { return nil }`
- `func (c *defaultConnect) Close() error { return nil }`
- `func (c *defaultConnect) List() (map[string]Job, error)`
- `func (c *defaultConnect) Add(name string, job Job) error`
- `func (c *defaultConnect) Enable(name string) error`
- `func (c *defaultConnect) Disable(name string) error`
- `func (c *defaultConnect) Remove(name string) error`
- `func (c *defaultConnect) AppendLog(log Log) error`
- `func (c *defaultConnect) History(jobName string, offset, limit int) (int64, []Log, error)`
- `func (c *defaultConnect) Lock(key string, ttl time.Duration) (bool, error)`
- `func Add(name string, job Job) error`
- `func Remove(name string) error`
- `func Enable(name string) error`
- `func Disable(name string) error`
- `func ListJobs() map[string]Job`
- `func RegisterJobs(jobs Jobs)`
- `func RegisterJob(name string, job Job)`
- `func RegisterConfig(config Config)`
- `func RegisterDriver(name string, driver Driver)`
- `func ListLogs(jobName string, offset, limit int) (int64, []Log)`
- `func (m *Module) Register(name string, value Any)`
- `func (m *Module) RegisterDriver(name string, driver Driver)`
- `func (m *Module) RegisterConfig(config Config)`
- `func (m *Module) RegisterJob(name string, job Job)`
- `func (m *Module) RegisterJobs(jobs Jobs)`
- `func (m *Module) Config(global Map)`
- `func (m *Module) Setup()`
- `func (m *Module) Open()`
- `func (m *Module) Start()`
- `func (m *Module) Stop()`
- `func (m *Module) Close()`
- `func (m *Module) Upsert(name string, job Job) error`
- `func (m *Module) Delete(name string) error`
- `func (m *Module) Enable(name string) error`
- `func (m *Module) Disable(name string) error`
- `func (m *Module) ListLogs(jobName string, offset, limit int) (int64, []Log)`

## 排错

- 模块未运行：确认空导入已存在
- driver 无效：确认驱动包已引入
- 配置不生效：检查配置段名是否为 `[cron]`
