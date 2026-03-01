# cron

`cron` 是 infrago 的**模块**。

## 包定位

- 类型：模块
- 作用：定时任务模块，负责任务注册、调度与执行。

## 主要功能

- 对上提供统一模块接口
- 对下通过驱动接口接入具体后端
- 支持按配置切换驱动实现

## 快速接入

```go
import _ "github.com/infrago/cron"
```

```toml
[cron]
driver = "default"
```

## 驱动实现接口列表

以下接口由驱动实现（来自模块 `driver.go`）：

### Driver

- `Connection(*Instance) (Connection, error)`

### Connection

- `Open() error`
- `Close() error`
- `Add(name string, job Job) error`
- `Enable(name string) error`
- `Disable(name string) error`
- `Remove(name string) error`
- `List() (map[string]Job, error)`
- `AppendLog(log Log) error`
- `History(jobName string, offset, limit int) (int64, []Log, error)`
- `Lock(key string, ttl time.Duration) (bool, error)`

## 全局配置项（所有配置键）

配置段：`[cron]`

- 未检测到配置键（请查看模块源码的 configure 逻辑）

## 说明

- `setting` 一般用于向具体驱动透传专用参数
- 多实例配置请参考模块源码中的 Config/configure 处理逻辑
