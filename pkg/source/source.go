package source

import (
	"context"
	"sync"
	"time"
)

// Source 定义代理源的基本接口
type Source interface {
	// Name 返回代理源的名称
	Name() string

	// Fetch 从源获取代理列表
	Fetch(ctx context.Context) (<-chan string, error)

	// IsAvailable 检查源是否可用
	IsAvailable() bool
	QueryTimeout() int
	ValidateLastFetchTime() bool
}

// BaseSource 提供基础实现
type BaseSource struct {
	name          string
	available     bool
	timeout       int
	lastFetchTime time.Time
	mu            sync.RWMutex
}

func (b *BaseSource) ValidateLastFetchTime() bool {
	return time.Since(b.lastFetchTime) > time.Duration(b.timeout)*time.Minute
}

// NewBaseSource 创建基础源
func NewBaseSource(name string, timeout int) *BaseSource {
	return &BaseSource{
		name:      name,
		timeout:   timeout,
		available: true, // 默认可用
	}
}

func (b *BaseSource) Name() string {
	return b.name
}

func (b *BaseSource) QueryTimeout() int {
	return b.timeout
}

func (b *BaseSource) IsAvailable() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.available
}

// SetAvailable 设置源可用状态(线程安全)
func (b *BaseSource) SetAvailable(available bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.available = available
}
