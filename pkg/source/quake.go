package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"io"
	"net/http"
	"time"
)

type QuakeSource struct {
	*BaseSource
	apiKey   string
	endpoint string
	maxSize  int
	query    string
}

func NewQuakeSource(apiKey, endpoint, query string, maxSize int, timeout int) *QuakeSource {
	return &QuakeSource{
		BaseSource: NewBaseSource("Quake", timeout),
		apiKey:     apiKey,
		endpoint:   endpoint,
		query:      query,
		maxSize:    maxSize,
	}
}

func (q *QuakeSource) Fetch(ctx context.Context) (<-chan string, error) {
	proxyChan := make(chan string)

	q.lastFetchTime = time.Now()

	go func() {
		defer func() {
			//fmt.Println("关闭 proxyChan")
			close(proxyChan)
		}()

		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 60 * time.Second,
			},
		}

		start := 1
		size := 10
		totalFetched := 0
		startTime := time.Now().AddDate(0, 0, -7).Format("2006-01-02") // 7天前

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 直接使用 q.endpoint（假设已包含完整路径）
				reqUrl := q.endpoint
				data := map[string]interface{}{
					"query":        q.query,
					"start":        start,
					"size":         size,
					"ignore_cache": true,
					"start_time":   startTime,
					"include": []string{
						"ip", "port",
					},
					"latest": true,
				}
				body, _ := json.MarshalIndent(data, "", "  ")

				req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, bytes.NewBuffer(body))
				if err != nil {
					return
				}
				req.Header.Set("X-QuakeToken", q.apiKey)
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					return
				}
				respBody, _ := io.ReadAll(resp.Body)
				//println(string(respBody))
				// ...保持原有响应解析逻辑...
				var result struct {
					Code    interface{} `json:"code"`
					Message string      `json:"message"`
					Data    interface{} `json:"data"`
				}

				if err := json.NewDecoder(bytes.NewBuffer(respBody)).Decode(&result); err != nil {
					resp.Body.Close()
					//q.SetAvailable(false)
					return
				}
				resp.Body.Close()

				_, ok := result.Code.(string)
				if ok {
					gologger.Warning().Msgf("query quake error: %v", result.Message)
					//q.SetAvailable(false)
					return
				}
				resultData, ok := result.Data.([]interface{})
				if !ok {
					return
				}

				for _, item := range resultData {
					resultData_, _ := item.(map[string]interface{})
					select {
					case <-ctx.Done():
						return
					case proxyChan <- fmt.Sprintf("socks5://%s:%v", resultData_["ip"], resultData_["port"]):
						totalFetched++
					}

					if q.maxSize > 0 && totalFetched >= q.maxSize {
						return
					}
				}

				if len(resultData) < size {
					return
				}

				start = start + size
				time.Sleep(5 * time.Second)
			}
		}
	}()

	return proxyChan, nil
}
