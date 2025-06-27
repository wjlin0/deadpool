package runner

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"github.com/remeh/sizedwaitgroup"
	"github.com/wjlin0/deadpool/pkg/source"
	"github.com/wjlin0/deadpool/pkg/types"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProxyInfo 存储单个代理的详细信息
type ProxyInfo struct {
	URL         string        `json:"url,omitempty"`      // 完整代理URL (socks5://user:pass@ip:port)
	IP          string        `json:"ip,omitempty"`       // 代理服务器IP
	Port        int           `json:"port,omitempty"`     // 代理服务器端口
	LastChecked time.Time     `json:"last_checked"`       // 最后检测时间（RFC3339格式）
	IsAlive     bool          `json:"is_alive,omitempty"` // 是否存活
	Latency     time.Duration `json:"latency,omitempty"`  // 延迟（纳秒数，可转换为毫秒）
	Username    string        `json:"username,omitempty"` // 认证用户名
	Password    string        `json:"password,omitempty"` // 认证密码（建议在前端脱敏）
	Source      string        `json:"source,omitempty"`   // 代理来源标识（如：file/hunter/quake）
	ExitIP      string        `json:"exit_ip,omitempty"`  // 出口IP（通过代理访问外部服务时显示的IP）
}

type IPGeoResponse struct {
	IP string `json:"ip"`
}

// SocksProxyManager 管理SOCKS代理
type SocksProxyManager struct {
	config       *types.ConfigOptions
	proxyMap     map[string]*ProxyInfo // 使用URL作为key的map
	mu           sync.RWMutex
	lastProxyURL string
	sources      []source.Source
}

// NewSocksProxyManager 创建新的代理管理器
func NewSocksProxyManager(cfg *types.ConfigOptions) *SocksProxyManager {
	spm := &SocksProxyManager{
		config:   cfg,
		proxyMap: make(map[string]*ProxyInfo),
	}
	var sources []source.Source
	// 初始化 文件源
	if cfg.SourcesConfig.File.Enabled {
		sources = append(sources, source.NewFileSource(cfg.SourcesConfig.File.Path, cfg.SourcesConfig.File.QueryTimeout))
	}
	if cfg.SourcesConfig.Hunter.Enabled {
		sources = append(sources, source.NewHunterSource(cfg.SourcesConfig.Hunter.APIKey, cfg.SourcesConfig.Hunter.Endpoint, cfg.SourcesConfig.Hunter.Query, cfg.SourcesConfig.Hunter.MaxSize, cfg.SourcesConfig.Hunter.QueryTimeout))
	}

	if cfg.SourcesConfig.CheckerProxy.Enabled {
		sources = append(sources, source.NewCheckerProxySource(cfg.SourcesConfig.CheckerProxy.Endpoint, cfg.SourcesConfig.CheckerProxy.QueryTimeout))
	}
	if cfg.SourcesConfig.Quake.Enabled {
		sources = append(sources, source.NewQuakeSource(cfg.SourcesConfig.Quake.APIKey, cfg.SourcesConfig.Quake.Endpoint, cfg.SourcesConfig.Quake.Query, cfg.SourcesConfig.Quake.MaxSize, cfg.SourcesConfig.Quake.QueryTimeout))
	}

	spm.sources = sources
	return spm
}

// NewSocksProxyManagerWithFile 从文件创建代理管理器
func NewSocksProxyManagerWithFile(cfg *types.ConfigOptions, filename string) (*SocksProxyManager, error) {

	m := NewSocksProxyManager(cfg)
	if err := m.LoadFromFile(filename); err != nil {
		return nil, err
	}

	return m, nil
}

// NextProxy 获取下一个可用代理(轮询方式)
func (m *SocksProxyManager) NextProxy() *ProxyInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxyMap) == 0 {
		return nil
	}

	// 创建一个有序的URL列表用于轮询
	var urls []string
	for u := range m.proxyMap {

		urls = append(urls, u)
	}

	// 查找上次返回的位置
	var lastIndex int = -1
	if m.lastProxyURL != "" {
		for i, u := range urls {
			if u == m.lastProxyURL {
				lastIndex = i
				break
			}
		}
	}

	// 从下一个位置开始查找可用代理
	startIndex := lastIndex + 1
	for i := 0; i < len(urls); i++ {
		currentIndex := (startIndex + i) % len(urls)
		p := m.proxyMap[urls[currentIndex]]
		if p.IsAlive {
			m.lastProxyURL = p.URL
			// 返回副本避免外部修改
			c := *p
			return &c
		}
	}

	// 没有可用代理
	return nil
}

// AddProxies 添加多个代理URL到管理器，并发进行存活检测并实时更新map
func (m *SocksProxyManager) AddProxies(proxyURLs []string, s string) {
	var wg sync.WaitGroup

	for _, proxyURL := range proxyURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			proxyInfo, err := parseProxyURL(url, s) // 忽略解析错误
			if err != nil {
				return
			}
			// 进行存活检测
			isAlive, latency := m.checkProxyAlive(proxyInfo)
			proxyInfo.IsAlive = isAlive
			proxyInfo.Latency = latency

			proxyInfo.LastChecked = time.Now()

			// 线程安全地立即添加代理到map
			m.mu.Lock()
			m.proxyMap[proxyInfo.URL] = proxyInfo
			m.mu.Unlock()
		}(proxyURL)
	}

	wg.Wait() // 等待所有检测完成
}

// AddProxiesWithAutoSave 添加代理并自动保存到文件
func (m *SocksProxyManager) AddProxiesWithAutoSave(proxyURLs []string, filename string, s string) error {
	m.AddProxies(proxyURLs, s)
	return m.SaveToFile(filename)
}

// StartAutoSave 启动定期自动保存
func (m *SocksProxyManager) StartAutoSave() {
	ticker := time.NewTicker(5)

	go func() {
		for range ticker.C {
			m.SaveToFile(m.config.Options.AliveDataPath)
		}
	}()
}

// LoadFromFile 从JSON文件加载代理信息，处理文件不存在的情况
func (m *SocksProxyManager) LoadFromFile(filename string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// 文件不存在，创建空文件
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(filename, []byte("{}"), 0644); err != nil {
			return fmt.Errorf("failed to create empty proxy file: %v", err)
		}
		return nil // 新创建的空文件，无需加载
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read proxy file: %v", err)
	}

	// 处理空文件情况
	if len(data) == 0 {
		data = []byte("{}") // 设置为空JSON对象
	}

	// 解析JSON数据
	var proxyInfos map[string]*ProxyInfo
	if err := json.Unmarshal(data, &proxyInfos); err != nil {
		// 尝试修复可能的损坏JSON（简单处理）
		if repaired, ok := tryRepairJSON(data); ok {
			if err := json.Unmarshal(repaired, &proxyInfos); err != nil {
				return fmt.Errorf("failed to parse proxy JSON (even after repair attempt): %v", err)
			}
		} else {
			return fmt.Errorf("failed to parse proxy JSON: %v", err)
		}
	}

	// 将解析的数据复制到proxyMap

	for u, pi := range proxyInfos {
		// 确保URL与key一致
		go m.AddProxy(u, pi.Source)
	}

	return nil
}

// SaveToFile 将代理信息以JSON格式保存到文件，确保原子性写入
func (m *SocksProxyManager) SaveToFile(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 将proxyMap转换为JSON
	data, err := json.MarshalIndent(m.proxyMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal proxies to JSON: %v", err)
	}

	// 原子性写入：先写入临时文件，然后重命名
	tempFile := filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp proxy file: %v", err)
	}

	// 重命名临时文件（原子性操作）
	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temp proxy file: %v", err)
	}

	return nil
}

// tryRepairJSON 尝试修复可能损坏的JSON数据
func tryRepairJSON(data []byte) ([]byte, bool) {
	str := strings.TrimSpace(string(data))
	if str == "" {
		return []byte("{}"), true
	}

	// 简单修复：如果以{开头但不以}结尾
	if strings.HasPrefix(str, "{") && !strings.HasSuffix(str, "}") {
		return []byte(str + "}"), true
	}

	// 简单修复：如果以[开头但不以]结尾（处理可能的数组格式）
	if strings.HasPrefix(str, "[") && !strings.HasSuffix(str, "]") {
		return []byte(str + "]"), true
	}

	return nil, false
}

// checkProxyAlive 检测SOCKS5代理是否存活
func (m *SocksProxyManager) checkProxyAlive(proxyInfo *ProxyInfo) (bool, time.Duration) {
	timeout := time.Duration(m.config.CheckSock.CheckInterval) * time.Second
	start := time.Now()

	// 1. 创建SOCKS5拨号器
	dialer, err := proxy.SOCKS5("tcp",
		net.JoinHostPort(proxyInfo.IP, strconv.Itoa(proxyInfo.Port)),
		&proxy.Auth{
			User:     proxyInfo.Username,
			Password: proxyInfo.Password,
		},
		&net.Dialer{Timeout: timeout})
	if err != nil {
		return false, 0
	}

	// 2. 创建HTTP客户端
	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: timeout,
	}

	// 4. 发送请求到检查URL
	for _, u := range m.config.CheckSock.CheckURL {
		resp, err := httpClient.Get(u)
		if err != nil {
			//gologger.Error().Msgf("%s:%s", proxyInfo.URL, err.Error())
			continue
		}
		defer resp.Body.Close()
		//gologger.Info().Msgf("%s -> %s %s", proxyInfo.URL, u, resp.Status)

		// 5. 检查关键词 (如果配置了)
		if len(m.config.CheckSock.CheckRspKeywords) != 0 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			for _, keyword := range m.config.CheckSock.CheckRspKeywords {
				if keyword == "" {
					return true, time.Since(start)
				}
				if strings.Contains(string(body), keyword) {
					return true, time.Since(start)
				}
			}
			return false, 0
		}

		return true, time.Since(start)
	}
	return true, time.Since(start)
}

func (m *SocksProxyManager) checkGeolocate(proxyInfo *ProxyInfo) bool {
	// 1. 检查功能开关
	if !m.config.CheckGeolocate.Enabled {
		return true
	}
	for _, u := range m.config.CheckGeolocate.CheckURL {
		timeout := time.Duration(m.config.CheckGeolocate.CheckInterval) * time.Second
		// 2. 创建代理客户端
		dialer, err := proxy.SOCKS5("tcp",
			net.JoinHostPort(proxyInfo.IP, strconv.Itoa(proxyInfo.Port)),
			&proxy.Auth{
				User:     proxyInfo.Username,
				Password: proxyInfo.Password,
			},
			&net.Dialer{Timeout: timeout})
		if err != nil {
			gologger.Warning().Msgf("%s : %s", proxyInfo.URL, err)
			continue
		}

		// 3. 发起请求
		client := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{Dial: dialer.Dial,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

		resp, err := client.Do(req)

		//resp, err := (&http.Client{
		//	Transport: &http.Transport{Dial: dialer.Dial,
		//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//	},
		//	Timeout: time.Duration(m.config.CheckSock.CheckInterval) * time.Second,
		//}).Get(m.config.CheckGeolocate.CheckURL)
		if err != nil {
			gologger.Warning().Msgf("%s : %s", proxyInfo.URL, err)

			continue
		}
		defer resp.Body.Close()
		//gologger.Info().Msg(fmt.Sprintf("%s -> %s %s", proxyInfo.URL, u, resp.Status))
		// 4. 读取响应
		body, _ := io.ReadAll(resp.Body) // 不处理读取错误，失败即返回false
		responseText := string(body)

		// 5. 执行关键词检查
		// 5. 执行关键词检查
		if len(m.config.CheckGeolocate.IncludeKeywords) > 0 {
			includeMatched := false
			if strings.ToLower(m.config.CheckGeolocate.IncludeKeywordCondition) == "and" {
				// AND 逻辑：必须匹配所有包含关键词
				includeMatched = true
				for _, kw := range m.config.CheckGeolocate.IncludeKeywords {
					if !strings.Contains(responseText, kw) {
						includeMatched = false
						break
					}
				}
			} else {
				// 默认 OR 逻辑：匹配任一包含关键词
				for _, kw := range m.config.CheckGeolocate.IncludeKeywords {
					if strings.Contains(responseText, kw) {
						includeMatched = true
						break
					}
				}
			}
			if !includeMatched {
				return false
			}
		}

		if len(m.config.CheckGeolocate.ExcludeKeywords) > 0 {
			excludeMatched := false
			if strings.ToLower(m.config.CheckGeolocate.ExcludeKeywordCondition) == "and" {
				// AND 逻辑：必须匹配所有排除关键词才排除
				excludeMatched = true
				for _, kw := range m.config.CheckGeolocate.ExcludeKeywords {
					if !strings.Contains(responseText, kw) {
						excludeMatched = false
						break
					}
				}
			} else {
				// 默认 OR 逻辑：匹配任一排除关键词就排除
				for _, kw := range m.config.CheckGeolocate.ExcludeKeywords {
					if strings.Contains(responseText, kw) {
						excludeMatched = true
						break
					}
				}
			}
			if excludeMatched {
				return false
			}
		}

		// 通过 JSON 序列化得到出口IP
		var result IPGeoResponse
		if err := json.Unmarshal([]byte(responseText), &result); err != nil {
			return false
		}

		m.mu.Lock()
		defer m.mu.Unlock()
		proxyInfo.ExitIP = result.IP
		return true
	}
	return false
}

func (m *SocksProxyManager) Dial(network, addr string) (net.Conn, error) {
	return m.DialContext(context.Background(), network, addr)
}

// DialContext 简化的拨号实现，不自动标记代理状态
// DialContext 完全支持上下文的实现
func (m *SocksProxyManager) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// 1. 获取代理
	proxyInfo := m.NextProxy()
	if proxyInfo == nil {
		log.Println("连接失败：没有可用代理")
		return nil, fmt.Errorf("no available proxies")
	}

	// 2. 创建拨号器
	baseDialer := &net.Dialer{
		Timeout:   time.Duration(m.config.CheckSock.CheckInterval)*time.Second + proxyInfo.Latency,
		KeepAlive: 30 * time.Second,
	}

	// 3. 创建SOCKS5拨号器
	socksDialer, err := proxy.SOCKS5(
		"tcp",
		net.JoinHostPort(proxyInfo.IP, strconv.Itoa(proxyInfo.Port)),
		&proxy.Auth{
			User:     proxyInfo.Username,
			Password: proxyInfo.Password,
		},
		baseDialer,
	)
	if err != nil {
		gologger.Error().Msgf("failed to create SOCKS5 dialer: %v", err)
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	// 4. 尝试连接
	var conn net.Conn
	if cd, ok := socksDialer.(interface {
		DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	}); ok {
		conn, err = cd.DialContext(ctx, network, addr)
	} else {
		conn, err = m.dialWithContext(ctx, socksDialer, network, addr)
	}

	format := ""
	var args []interface{}

	// 5. 记录连接结果
	if err != nil {
		format = "error -> %s -> %v"
		args = append(args, proxyInfo.URL)
		args = append(args, err)
		gologger.Info().Msgf(format, args...)
		return nil, err
	} else {
		exitIP := m.getExitIP(proxyInfo)
		remoteAddr := conn.RemoteAddr().String()
		localAddr := conn.LocalAddr().String()
		format = "success -> %s -> %s -> %s"
		args = append(args, remoteAddr)
		args = append(args, localAddr)
		args = append(args, exitIP)
		gologger.Info().Msgf(format, args...)
	}

	return conn, err
}

// 辅助方法：获取出口IP
func (m *SocksProxyManager) getExitIP(proxyInfo *ProxyInfo) string {
	if proxyInfo.ExitIP != "" {
		return proxyInfo.ExitIP
	}
	return proxyInfo.IP
}

// 辅助方法：带上下文的手动拨号
func (m *SocksProxyManager) dialWithContext(ctx context.Context, dialer proxy.Dialer, network, addr string) (net.Conn, error) {
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := dialer.Dial(network, addr)
		if err != nil {
			errChan <- err
			return
		}
		select {
		case <-ctx.Done():
			conn.Close()
			errChan <- ctx.Err()
		default:
			connChan <- conn
		}
	}()

	select {
	case <-ctx.Done():
		go func() { <-connChan }() // 清理可能的成功连接
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case conn := <-connChan:
		return conn, nil
	}
}

// parseProxyURL 解析代理URL为ProxyInfo结构体
func parseProxyURL(proxyURL string, s string) (*ProxyInfo, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	// 提取IP和端口
	host := u.Hostname()
	portStr := u.Port()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	// 提取用户名和密码
	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}

	return &ProxyInfo{
		URL:      proxyURL,
		IP:       host,
		Port:     port,
		Username: username,
		Password: password,
		IsAlive:  false, // 默认设为不存活
		Source:   s,
	}, nil
}

func (m *SocksProxyManager) AddProxy(proxyURL string, s string) {

	// 判断 源是否存在 是否在 proxyMap 中
	m.mu.RLock()
	proxyInfo, ok := m.proxyMap[proxyURL]
	if ok && proxyInfo.IsAlive {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	proxyInfo, err := parseProxyURL(proxyURL, s)
	if err != nil {
		return
	}

	if !m.checkGeolocate(proxyInfo) {
		return
	}

	isAlive, latency := m.checkProxyAlive(proxyInfo)
	proxyInfo.IsAlive = isAlive
	proxyInfo.Latency = latency
	proxyInfo.LastChecked = time.Now()
	// 如果超过 5秒的延迟 就不要了

	if isAlive && latency < 5*time.Second {
		gologger.Info().Msgf("代理可用: %s", proxyInfo.URL)
		m.mu.Lock()
		m.proxyMap[proxyInfo.URL] = proxyInfo
		m.mu.Unlock()
	}
}

// StartAutoCheck 启动自动存活检测
func (m *SocksProxyManager) StartAutoCheck() {
	go func() {
		for {
			m.mu.RLock()
			var wg sizedwaitgroup.SizedWaitGroup
			wg = sizedwaitgroup.New(m.config.CheckSock.MaxConcurrentReq)

			for _, p := range m.proxyMap {
				if !m.shouldCheckNow(p, time.Now()) {
					continue
				}

				wg.Add()
				go func(proxy *ProxyInfo) {
					defer wg.Done()

					isAlive, latency := m.checkProxyAlive(proxy)

					m.mu.Lock()
					proxy.IsAlive = isAlive
					proxy.Latency = latency
					proxy.LastChecked = time.Now()
					m.mu.Unlock()
				}(p)
			}

			m.mu.RUnlock()
			wg.Wait()

			// 短暂休眠避免CPU空转
			time.Sleep(1 * time.Second)
		}
	}()
}

// shouldCheckNow 判断是否需要立即检测
func (m *SocksProxyManager) shouldCheckNow(p *ProxyInfo, now time.Time) bool {
	if !p.IsAlive {
		return true // 不可用的代理优先检测
	}

	var interval time.Duration
	switch p.Source {
	case "file":
		interval = time.Duration(m.config.SourcesConfig.File.CheckInterval) * time.Second
	case "hunter":
		interval = time.Duration(m.config.SourcesConfig.Hunter.CheckInterval) * time.Second
	case "quake":
		interval = time.Duration(m.config.SourcesConfig.Quake.CheckInterval) * time.Second
	case "checkerProxy":
		interval = time.Duration(m.config.SourcesConfig.CheckerProxy.CheckInterval) * time.Second
	default:
		interval = time.Duration(m.config.CheckSock.CheckInterval) * time.Second
	}

	return now.Sub(p.LastChecked) > interval
}

func (m *SocksProxyManager) StartAutoSource() {
	wg := sizedwaitgroup.New(m.config.CheckSock.MaxConcurrentReq)
	wg2 := sizedwaitgroup.New(4)
	go func() {
		for {
			m.mu.RLock()
			if len(m.proxyMap) >= m.config.CheckSock.MinSize {
				m.mu.RUnlock()
				continue
			}
			m.mu.RUnlock()

			for _, s := range m.sources {
				if !s.ValidateLastFetchTime() {
					continue
				}
				if !s.IsAvailable() {
					continue
				}

				wg2.Add()
				go func(s source.Source) {
					defer wg2.Done()
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					proxyChan, err := s.Fetch(ctx)
					if err != nil {
						return
					}
					for p := range proxyChan {
						m.mu.RLock()
						if _, ok := m.proxyMap[p]; ok {
							m.mu.RUnlock()
							continue
						}
						m.mu.RUnlock()
						gologger.Warning().Msgf("%s 获得 %s 正在检测代理可用性", s.Name(), p)

						wg.Add()
						go func(proxy string) {
							defer wg.Done()
							m.AddProxy(proxy, s.Name())
						}(p)

						m.mu.RLock()
						if len(m.proxyMap) >= m.config.CheckSock.MinSize {
							m.mu.RUnlock()
							cancel()
							return
						}
						m.mu.RUnlock()
					}

					wg.Wait()

				}(s)
			}
			wg2.Wait()

		}
	}()
}
func (m *SocksProxyManager) Start() func(network, addr string) (net.Conn, error) {
	// 开启 自动保存文件
	m.StartAutoSave()
	// 开启 自动获取代理
	m.StartAutoSource()

	// 开启 自动存活检测
	m.StartAutoCheck()
	return m.Dial
}

func (m *SocksProxyManager) StartContext() func(ctx context.Context, network, addr string) (net.Conn, error) {
	// 开启 自动保存文件
	m.StartAutoSave()
	// 开启 自动获取代理
	m.StartAutoSource()

	// 开启 自动存活检测
	m.StartAutoCheck()
	return m.DialContext

}
