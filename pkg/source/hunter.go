package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"net/http"
	"net/url"
	"time"
)

type HunterSource struct {
	*BaseSource
	apiKey   string
	endpoint string
	maxSize  int
	query    string
}

func NewHunterSource(apiKey, endpoint, query string, maxSize int, timeout int) *HunterSource {
	return &HunterSource{
		BaseSource: NewBaseSource("hunter", timeout),
		apiKey:     apiKey,
		endpoint:   endpoint,
		query:      query,
		maxSize:    maxSize,
	}
}

func (h *HunterSource) Fetch(ctx context.Context) (<-chan string, error) {
	proxyChan := make(chan string)

	h.lastFetchTime = time.Now()

	go func() {
		defer func() {
			fmt.Println("关闭 proxyChan")
			close(proxyChan)
		}()

		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 60 * time.Second,
			},
		}

		page := 1
		pageSize := 50
		totalFetched := 0
		startTime := time.Now().AddDate(0, 0, -1).Format("2006-01-02") // 7天前

		// 强制SOCKS5协议查询
		baseQuery := `protocol="socks5"`
		if h.query != "" {
			baseQuery = h.query
		}
		encodedSearch := base64.URLEncoding.EncodeToString([]byte(baseQuery))

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 直接使用 h.endpoint（假设已包含完整路径）
				reqUrl := fmt.Sprintf("%s?search=%s&page=%d&page_size=%d&api-key=%s&start_time=%s",
					h.endpoint, // 示例: "https://api.hunter.io/openApi/search"
					encodedSearch,
					page,
					pageSize,
					url.QueryEscape(h.apiKey),
					startTime,
				)

				gologger.Info().Msg(reqUrl)

				req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
				if err != nil {
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					return
				}

				// ...保持原有响应解析逻辑...
				var result struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Data    struct {
						Arr []struct {
							IP       string `json:"ip"`
							Port     int    `json:"port"`
							Protocol string `json:"protocol"`
						} `json:"arr"`
					} `json:"data"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					resp.Body.Close()
					//h.SetAvailable(false)
					return
				}
				resp.Body.Close()

				if result.Code != 200 {
					h.SetAvailable(false)
					return
				}

				for _, item := range result.Data.Arr {
					select {
					case <-ctx.Done():
						return
					case proxyChan <- fmt.Sprintf("socks5://%s:%d", item.IP, item.Port):
						totalFetched++
					}

					if h.maxSize > 0 && totalFetched >= h.maxSize {
						return
					}
				}

				if len(result.Data.Arr) < pageSize {
					return
				}

				page++
				time.Sleep(5 * time.Second)
			}
		}
	}()

	return proxyChan, nil
}
