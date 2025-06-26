package source

import (
	"bufio"
	"context"
	"os"
	"strings"
	"time"
)

// FileSource 实现Source接口
type FileSource struct {
	*BaseSource
	filePath string
}

// NewFileSource 创建新的文件源
func NewFileSource(filePath string, timeout int) *FileSource {
	return &FileSource{
		BaseSource: NewBaseSource("file", timeout),
		filePath:   filePath,
	}
}

// Name 返回源名称
func (f *FileSource) Name() string {
	return f.name
}

// Fetch 从文件读取代理列表，返回一个通道
func (f *FileSource) Fetch(ctx context.Context) (<-chan string, error) {
	f.lastFetchTime = time.Now()
	file, err := os.Open(f.filePath)
	if err != nil {
		return nil, err
	}

	proxyChan := make(chan string)

	go func() {
		defer close(proxyChan)
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			select {
			case <-ctx.Done(): // 监听取消信号
				return
			default:
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					proxyChan <- line
				}
			}
		}
	}()
	return proxyChan, nil
}

// IsAvailable 检查源是否可用
func (f *FileSource) IsAvailable() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.available
}

// setAvailable 设置源可用状态(内部方法)
func (f *FileSource) setAvailable(available bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.available = available
}
