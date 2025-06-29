package source

import (
	"bytes"
	"context"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/retryablehttp-go"
	"github.com/tidwall/gjson"
	"github.com/wjlin0/deadpool/pkg/types"
	"io"
	"strconv"
	"strings"
	"time"
)

type CustomSource struct {
	*BaseSource
	endpoint     string
	method       string
	headers      map[string]string
	body         string
	proxyConfig  *types.ProxyExtractConfig
	responseType string // "json" 或 "text"
	enablePaging bool
	currentPage  int
	pageSize     int
	maxSize      int
}

func NewCustomSource(
	customName string,
	endpoint string,
	method string,
	headers map[string]string,
	body string,
	proxyConfig *types.ProxyExtractConfig,
	responseType string,
	timeout int,
	enablePaging bool,
	maxSize int,
) *CustomSource {
	return &CustomSource{
		BaseSource:   NewBaseSource(customName, timeout),
		endpoint:     endpoint,
		method:       method,
		headers:      headers,
		body:         body,
		proxyConfig:  proxyConfig,
		responseType: responseType,
		enablePaging: enablePaging,
		currentPage:  1,
		pageSize:     10,
		maxSize:      maxSize,
	}
}

func (c *CustomSource) Fetch(ctx context.Context) (<-chan string, error) {
	proxyChan := make(chan string)
	c.lastFetchTime = time.Now()
	go func() {
		defer func() {
			gologger.Debug().Msgf("关闭%s自定义代理通道", c.Name())
			close(proxyChan)
		}()

		defaultOptions := retryablehttp.DefaultOptionsSingle
		client := retryablehttp.NewClient(defaultOptions)
		totalCount := 0
		for {
			endpoint := c.endpoint
			body := c.body

			// Handle pagination if enabled
			if c.enablePaging {
				// Replace {page} and {pageSize} in endpoint
				if strings.Contains(endpoint, "{page}") {
					endpoint = strings.ReplaceAll(endpoint, "{page}", strconv.Itoa(c.currentPage))
				}
				if strings.Contains(endpoint, "{pageSize}") {
					endpoint = strings.ReplaceAll(endpoint, "{pageSize}", strconv.Itoa(c.pageSize))
				}

				// Replace {page} and {pageSize} in body
				if strings.Contains(body, "{page}") {
					body = strings.ReplaceAll(body, "{page}", strconv.Itoa(c.currentPage))
				}
				if strings.Contains(body, "{pageSize}") {
					body = strings.ReplaceAll(body, "{pageSize}", strconv.Itoa(c.pageSize))
				}
			}

			req, err := retryablehttp.NewRequestWithContext(ctx, c.method, endpoint, strings.NewReader(body))
			if err != nil {
				gologger.Warning().Msgf("创建请求失败: %v", err)
				return
			}

			for key, value := range c.headers {
				req.Header.Set(key, value)
			}

			gologger.Info().Msgf("发送请求: %s %s (Page: %d)", c.method, endpoint, c.currentPage)

			resp, err := client.Do(req)
			if err != nil {
				gologger.Warning().Msgf("请求失败: %v", err)
				return
			}

			var proxyCount int
			switch c.responseType {
			case "json":
				proxyCount = c.extractProxiesFromJSON(resp.Body, proxyChan, ctx)
			case "text":
				proxyCount = c.extractProxiesFromText(resp.Body, proxyChan, ctx)
			case "xpath":
				proxyCount = c.extractProxiesFromXpath(resp.Body, proxyChan, ctx)
			default:
				gologger.Warning().Msgf("不支持的响应类型: %s", c.responseType)
				resp.Body.Close()
				return
			}
			resp.Body.Close()

			// If pagination is disabled or no proxies were found in this page, break the loop
			if !c.enablePaging || proxyCount == 0 || totalCount >= c.maxSize {
				break
			}
			totalCount += proxyCount
			// Move to next page
			c.currentPage++
		}
	}()

	return proxyChan, nil
}

// extractProxiesFromJSON 从 JSON 响应中提取代理
// 返回提取到的代理数量
func (c *CustomSource) extractProxiesFromJSON(body io.Reader, proxyChan chan<- string, ctx context.Context) int {
	data, err := io.ReadAll(body)
	if err != nil {
		gologger.Warning().Msgf("读取响应失败: %v", err)
		return 0
	}
	proxyList := gjson.GetBytes(data, c.proxyConfig.ProxyListPath)
	if !proxyList.Exists() {
		gologger.Warning().Msgf("未找到代理列表，路径: %s", c.proxyConfig.ProxyListPath)
		return 0
	}

	count := 0
	proxyList.ForEach(func(_, proxy gjson.Result) bool {
		select {
		case <-ctx.Done():
			return false
		default:
			if proxy.Type == 3 {
				proxyStr := proxy.Str
				if strings.Contains(proxyStr, "://") {
					protocol := strings.Split(proxyStr, "://")[0]
					if protocol != "socks5" {
						return true
					}
					proxyStr = strings.Split(proxyStr, "://")[1]

				} else {
					proxyStr = fmt.Sprintf("socks5://%s", proxyStr)
				}

				proxyChan <- proxyStr
				count++
				return true
			} else {
				ip := proxy.Get(c.proxyConfig.IPField).String()
				port := proxy.Get(c.proxyConfig.PortField).String()

				username := proxy.Get(c.proxyConfig.UserField).String()
				password := proxy.Get(c.proxyConfig.PasswordField).String()

				if ip == "" || port == "" {
					gologger.Debug().Msgf("跳过无效代理: ip=%q, port=%q", ip, port)
					return true
				}
				proxyAddr := ""

				switch {
				case username == "" && password == "":
					proxyAddr = fmt.Sprintf("socks5://%s:%s", ip, port)
				default:
					proxyAddr = fmt.Sprintf("socks5://%s:%s@%s:%s", username, password, ip, port)

				}

				proxyChan <- proxyAddr
				count++
				return true
			}

		}
	})
	return count
}

// extractProxiesFromText 从文本响应中提取代理
// 返回提取到的代理数量
func (c *CustomSource) extractProxiesFromText(body io.Reader, proxyChan chan<- string, ctx context.Context) int {
	data, err := io.ReadAll(body)
	if err != nil {
		gologger.Warning().Msgf("读取响应失败: %v", err)
		return 0
	}

	count := 0
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		select {
		case <-ctx.Done():
			return count
		default:
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "://") {
				protocal := strings.Split(line, "://")[0]
				if protocal != "socks5" {
					gologger.Debug().Msgf("跳过非 socks5 代理: %s", line)
					continue
				}
				line = strings.Split(line, "://")[1]
			}

			proxyChan <- fmt.Sprintf("socks5://%s", line)
			count++
		}
	}
	return count
}

func (c *CustomSource) extractProxiesFromXpath(body io.Reader, proxyChan chan<- string, ctx context.Context) int {

	data, err := io.ReadAll(body)
	if err != nil {
		gologger.Warning().Msgf("读取响应失败: %v", err)
		return 0
	}
	pathNode, err := htmlquery.Parse(bytes.NewBuffer(data))
	if err != nil {
		gologger.Warning().Msgf("解析响应失败: %v", err)
		return 0
	}
	// fmt.Println(string(data))

	doc, err := htmlquery.QueryAll(pathNode, c.proxyConfig.ProxyListPath)
	if err != nil {
		gologger.Warning().Msgf("读取响应失败: %v", err)
		return 0
	}

	count := 0
	for _, node := range doc {
		select {
		case <-ctx.Done():
			return 0
		default:
			ipNode := htmlquery.FindOne(node, c.proxyConfig.IPField)
			if ipNode == nil {
				continue
			}
			ip := strings.TrimSpace(htmlquery.InnerText(ipNode))

			portNode := htmlquery.FindOne(node, c.proxyConfig.PortField)
			if portNode == nil {
				continue
			}
			port := strings.TrimSpace(htmlquery.InnerText(portNode))
			username := ""
			usernameNode := htmlquery.FindOne(node, c.proxyConfig.UserField)
			if usernameNode != nil {
				username = strings.TrimSpace(htmlquery.InnerText(usernameNode))
			}
			password := ""
			passwordNode := htmlquery.FindOne(node, c.proxyConfig.PasswordField)
			if passwordNode != nil {
				password = strings.TrimSpace(htmlquery.InnerText(passwordNode))
			}

			if ip == "" || port == "" {
				gologger.Debug().Msgf("跳过无效代理: ip=%q, port=%q", ip, port)
				continue
			}
			proxyAddr := ""

			switch {
			case username == "" && password == "":
				proxyAddr = fmt.Sprintf("socks5://%s:%s", ip, port)
			default:
				proxyAddr = fmt.Sprintf("socks5://%s:%s@%s:%s", username, password, ip, port)

			}

			proxyChan <- proxyAddr
			count++
		}
	}
	return count
}
