package source

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"net/http"
	"time"
)

type CheckerProxySource struct {
	*BaseSource
	endpoint string
}

func NewCheckerProxySource(endpoint string, timeout int) *CheckerProxySource {
	return &CheckerProxySource{
		BaseSource: NewBaseSource("CheckerProxy", timeout),
		endpoint:   endpoint,
	}
}

func (h *CheckerProxySource) Fetch(ctx context.Context) (<-chan string, error) {
	proxyChan := make(chan string)
	h.lastFetchTime = time.Now()

	go func() {
		defer close(proxyChan)

		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 60 * time.Second,
			},
		}

		totalFetched := 0
		maxDays := 5 // 最多回溯5天
		daysChecked := 0

		for daysChecked < maxDays {
			select {
			case <-ctx.Done():
				return
			default:
				// 计算当前检查的日期（从昨天开始往前推）
				checkDate := time.Now().AddDate(0, 0, -daysChecked).Format("2006-01-02")
				reqUrl := fmt.Sprintf("%s/%s", h.endpoint, checkDate)

				gologger.Info().Msgf("Fetching proxies for date: %s", reqUrl)

				req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
				if err != nil {
					gologger.Error().Msgf("Failed to create request: %v", err)
					daysChecked++
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					gologger.Error().Msgf("Request failed for date %s: %v", checkDate, err)
					daysChecked++
					continue
				}

				var result struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
					Data    struct {
						ProxyList []string `json:"proxyList"`
					} `json:"data"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					gologger.Error().Msgf("Failed to decode response for date %s: %v", checkDate, err)
					resp.Body.Close()
					daysChecked++
					continue
				}
				resp.Body.Close()

				if !result.Success {
					gologger.Error().Msgf("Unsuccessful response for date %s: %s", checkDate, result.Message)
					daysChecked++
					continue
				}

				// 发送获取到的代理
				for _, item := range result.Data.ProxyList {
					select {
					case <-ctx.Done():
						return
					case proxyChan <- fmt.Sprintf("socks5://%s", item):
						totalFetched++
					}
				}

				daysChecked++

			}
		}

		gologger.Info().Msgf("Finished checking %d days, total proxies fetched: %d", daysChecked, totalFetched)
	}()

	return proxyChan, nil
}
