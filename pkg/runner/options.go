package runner

import (
	"fmt"
	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/gologger"
	"github.com/wjlin0/deadpool/pkg/types"
	updateutils "github.com/wjlin0/utils/update"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

func ParserOptions() *types.Options {
	options := &types.Options{}
	set := goflags.NewFlagSet()
	set.SetDescription(fmt.Sprintf("deadpool %s  test", Version))
	set.CreateGroup("Input", "输入",
		set.StringVarP(&options.ConfigPath, "config", "c", "config.yaml", "配置文件"),
		set.StringVarP(&options.AliveDataPath, "alive-data-path", "adp", "aliveDataPath.json", "存储的存活IP列表"),
	)
	set.CreateGroup("Update", "更新",
		set.CallbackVar(updateutils.GetUpdateToolCallback(pathScanRepoName, Version), "update", "更新版本"),
		set.BoolVarP(&options.DisableUpdateCheck, "disable-update-check", "duc", false, "跳过自动检查更新"),
	)
	_ = set.Parse()
	// show banner
	showBanner()
	if !options.DisableUpdateCheck {
		latestVersion, err := updateutils.GetToolVersionCallback(toolName, pathScanRepoName)()
		if err != nil {
			gologger.Info().Msgf("Current %s version v%v ", toolName, Version)
		} else {
			gologger.Info().Msgf("Current %s version v%v %v", toolName, Version, updateutils.GetVersionDescription(Version, latestVersion))
		}

	} else {
		gologger.Info().Msgf("Current %s version v%v ", toolName, Version)
	}
	return options

}

func ParserConfigOptions(opts *types.Options) (*types.ConfigOptions, error) {
	// 1. 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(opts.ConfigPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	// 2. 检查文件是否存在
	if _, err := os.Stat(opts.ConfigPath); os.IsNotExist(err) {
		// 3. 文件不存在时创建默认配置（完全保持您的默认值）
		defaultConfig := &types.ConfigOptions{
			Listener: &types.Listener{
				IP:    "127.0.0.1", // 保持您的默认值
				Port:  1080,        // 保持您的默认值
				Auths: []string{},  // 保持您的默认值
			},
			CheckSock: &types.CheckSock{
				CheckURL:         []string{"https://www.baidu.com"}, // 保持您的默认值
				CheckRspKeywords: []string{"百度一下"},                  // 保持您的默认值
				MaxConcurrentReq: 100,                               // 保持您的默认值
				CheckInterval:    60,                                // 保持您的默认值
				MinSize:          50,                                // 保持您的默认值
			},
			CheckGeolocate: &types.CheckGeolocate{
				CheckURL: []string{
					"https://qifu-api.baidubce.com/ip/local/geo/v1/district",
					"https://ipapi.co/json",
				}, // 保持您的默认值
				ExcludeKeywords:         []string{"澳门", "香港", "台湾"}, // 保持您的默认值
				IncludeKeywords:         []string{"中国"},             // 保持您的默认值
				IncludeKeywordCondition: "or",
				ExcludeKeywordCondition: "or",
			},
			SourcesConfig: &types.SourcesConfig{
				Hunter: &types.HunterSource{
					Enabled:       false,
					Endpoint:      "https://hunter.qianxin.com/openApi/search",
					Query:         "protocol==\"socks5\"&& protocol.banner=\"No authentication\"&&ip.country=\"CN\"",
					CheckInterval: 60,
					QueryTimeout:  60,
					MaxSize:       50,
				},
				Quake: &types.QuakeSource{
					Enabled:       false,
					Endpoint:      "https://quake.360.net/api/v3/search/quake_service",
					Query:         "service:socks5  AND country: \"CN\" AND response:\"No authentication\"",
					CheckInterval: 60,
					QueryTimeout:  60,
					MaxSize:       50,
				},
				File: &types.FileSource{
					Enabled:       false,
					Path:          "proxies.txt",
					CheckInterval: 60 * 5,
				},
				CheckerProxy: &types.CheckerProxy{
					Enabled:       true,
					Endpoint:      "https://api.checkerproxy.net/v1/landing/archive",
					CheckInterval: 60,
					QueryTimeout:  60 * 24,
				},
			},
			Options: opts,
		}

		// 原子写入默认配置
		if err := saveConfigToFile(opts.ConfigPath, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
		return defaultConfig, nil
	}

	// 4. 读取并解析现有配置文件（保持您原有的解析逻辑不变）
	data, err := os.ReadFile(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config types.ConfigOptions
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	// 5. 初始化配置（完全保持您的默认值设置逻辑）
	if config.Listener == nil {
		config.Listener = &types.Listener{}
	}
	if config.CheckSock == nil {
		config.CheckSock = &types.CheckSock{}
	}
	if config.CheckGeolocate == nil {
		config.CheckGeolocate = &types.CheckGeolocate{}
	}
	if config.SourcesConfig == nil {
		config.SourcesConfig = &types.SourcesConfig{}
	}

	// 设置Listener默认值
	if config.Listener.IP == "" {
		config.Listener.IP = "127.0.0.1"
	}
	if config.Listener.Port == 0 {
		config.Listener.Port = 1080
	}
	if config.Listener.Auths == nil {
		config.Listener.Auths = []string{}
	}

	// 设置CheckSock默认值
	if config.CheckSock.CheckURL == nil {
		config.CheckSock.CheckURL = []string{"https://www.baidu.com"}
	}
	if config.CheckSock.CheckRspKeywords == nil {
		config.CheckSock.CheckRspKeywords = []string{"百度一下"}
	}
	if config.CheckSock.MaxConcurrentReq == 0 {
		config.CheckSock.MaxConcurrentReq = 100
	}
	if config.CheckSock.CheckInterval == 0 {
		config.CheckSock.CheckInterval = 60
	}
	if config.CheckSock.MinSize == 0 {
		config.CheckSock.MinSize = 50
	}

	// 设置CheckGeolocate默认值
	if config.CheckGeolocate.CheckURL == nil {
		config.CheckGeolocate.CheckURL = []string{
			"https://qifu-api.baidubce.com/ip/local/geo/v1/district",
			"https://ipapi.co/json",
		}
	}
	if config.CheckGeolocate.ExcludeKeywords == nil {
		config.CheckGeolocate.ExcludeKeywords = []string{"澳门", "香港", "台湾", "HK", "TW"}
	}
	if config.CheckGeolocate.ExcludeKeywordCondition == "" {
		config.CheckGeolocate.ExcludeKeywordCondition = "or"
	}

	if config.CheckGeolocate.IncludeKeywords == nil {
		config.CheckGeolocate.IncludeKeywords = []string{"中国", "CN"}
	}
	if config.CheckGeolocate.IncludeKeywordCondition == "" {
		config.CheckGeolocate.IncludeKeywordCondition = "or"
	}

	// 初始化SourcesConfig子结构体
	if config.SourcesConfig.Hunter == nil {
		config.SourcesConfig.Hunter = &types.HunterSource{
			Enabled:       false,
			Endpoint:      "https://hunter.qianxin.com/openApi/search",
			Query:         "protocol==\"socks5\"&& protocol.banner=\"No authentication\"&&ip.country=\"CN\"",
			CheckInterval: 60,
			QueryTimeout:  60,
			MaxSize:       50,
		}
	} else {
		if config.SourcesConfig.Hunter.Endpoint == "" {
			config.SourcesConfig.Hunter.Endpoint = "https://hunter.qianxin.com/openApi/search"
		}
		if config.SourcesConfig.Hunter.Query == "" {
			config.SourcesConfig.Hunter.Query = "protocol==\"socks5\"&& protocol.banner=\"No authentication\"&&ip.country=\"CN\""
		}
		if config.SourcesConfig.Hunter.CheckInterval == 0 {
			config.SourcesConfig.Hunter.CheckInterval = 60
		}
		if config.SourcesConfig.Hunter.QueryTimeout == 0 {
			config.SourcesConfig.Hunter.QueryTimeout = 60
		}
		if config.SourcesConfig.Hunter.MaxSize == 0 {
			config.SourcesConfig.Hunter.MaxSize = 50
		}
	}

	if config.SourcesConfig.Quake == nil {
		config.SourcesConfig.Quake = &types.QuakeSource{
			Enabled:       false,
			Endpoint:      "https://quake.360.net/api/v3/search/quake_service",
			Query:         "service:socks5  AND country: \"CN\" AND response:\"No authentication\"",
			CheckInterval: 60,
			QueryTimeout:  60,
			MaxSize:       50,
		}
	} else {
		if config.SourcesConfig.Quake.Endpoint == "" {
			config.SourcesConfig.Quake.Endpoint = "https://quake.360.net/api/v3/search/quake_service"
		}
		if config.SourcesConfig.Quake.Query == "" {
			config.SourcesConfig.Quake.Query = "service:socks5  AND country: \"CN\" AND response:\"No authentication\""
		}
		if config.SourcesConfig.Quake.CheckInterval == 0 {
			config.SourcesConfig.Quake.CheckInterval = 60
		}
		if config.SourcesConfig.Quake.QueryTimeout == 0 {
			config.SourcesConfig.Quake.QueryTimeout = 60
		}
		if config.SourcesConfig.Quake.MaxSize == 0 {
			config.SourcesConfig.Quake.MaxSize = 50
		}
	}

	if config.SourcesConfig.File == nil {
		config.SourcesConfig.File = &types.FileSource{
			Enabled:       false,
			Path:          "proxies.txt",
			CheckInterval: 60 * 5,
		}
	} else {
		if config.SourcesConfig.File.Path == "" {
			config.SourcesConfig.File.Path = "proxies.txt"
		}
		if config.SourcesConfig.File.CheckInterval == 0 {
			config.SourcesConfig.File.CheckInterval = 60 * 5
		}
	}

	if config.SourcesConfig.CheckerProxy == nil {
		config.SourcesConfig.CheckerProxy = &types.CheckerProxy{
			Enabled:       true,
			Endpoint:      "https://api.checkerproxy.net/v1/landing/archive",
			CheckInterval: 60,
			QueryTimeout:  60 * 24,
		}
	} else {
		if config.SourcesConfig.CheckerProxy.Endpoint == "" {
			config.SourcesConfig.CheckerProxy.Endpoint = "https://api.checkerproxy.net/v1/landing/archive"
		}
		if config.SourcesConfig.CheckerProxy.CheckInterval == 0 {
			config.SourcesConfig.CheckerProxy.CheckInterval = 60
		}
		if config.SourcesConfig.CheckerProxy.QueryTimeout == 0 {
			config.SourcesConfig.CheckerProxy.QueryTimeout = 60 * 24
		}
	}

	config.Options = opts
	return &config, nil
}

// saveConfigToFile 原子化保存配置文件
func saveConfigToFile(path string, config *types.ConfigOptions) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// 原子写入：先写临时文件再重命名
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
