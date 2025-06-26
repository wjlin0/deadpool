package types

type Options struct {
	ConfigPath         string
	AliveDataPath      string
	DisableUpdateCheck bool
}

type ConfigOptions struct {
	Options        *Options        `yaml:"-"`
	Listener       *Listener       `yaml:"listener"`
	CheckSock      *CheckSock      `yaml:"checkSock"`
	CheckGeolocate *CheckGeolocate `yaml:"checkGeolocate"`
	SourcesConfig  *SourcesConfig  `yaml:"sourcesConfig"`
}

type SourcesConfig struct {
	Hunter       *HunterSource `yaml:"hunter"`
	Quake        *QuakeSource  `yaml:"quake"`
	File         *FileSource   `yaml:"file"`
	CheckerProxy *CheckerProxy `yaml:"checkerProxy"`
}

type HunterSource struct {
	Enabled       bool   `yaml:"enabled"`
	APIKey        string `yaml:"apiKey"`
	Endpoint      string `yaml:"endpoint"`
	Query         string `yaml:"query"`
	MaxSize       int    `yaml:"maxSize"`
	CheckInterval int    `yaml:"checkInterval"` // 检测间隔(分钟)

	QueryTimeout int `yaml:"queryTimeout"` // 请求延迟时间
}

type QuakeSource struct {
	Enabled       bool   `yaml:"enabled"`
	APIKey        string `yaml:"apiKey"`
	Endpoint      string `yaml:"endpoint"`
	MaxSize       int    `yaml:"maxSize"`
	Query         string `yaml:"query"`
	CheckInterval int    `yaml:"checkInterval"` // 检测间隔(分钟)

	QueryTimeout int `yaml:"queryTimeout"` // 请求延迟时间
}

type FileSource struct {
	Enabled       bool   `yaml:"enabled"`
	Path          string `yaml:"path"`
	CheckInterval int    `yaml:"checkInterval"` // 检测间隔(分钟)

	QueryTimeout int `yaml:"queryTimeout"` // 请求延迟时间
}

type CheckerProxy struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`
	CheckInterval int    `yaml:"checkInterval"` // 检测间隔(分钟)
	QueryTimeout  int    `yaml:"queryTimeout"`  // 请求延迟时间
}

type Listener struct {
	IP    string   `yaml:"ip"`
	Port  int      `yaml:"port"`
	Auths []string `yaml:"auths"`
}

type CheckSock struct {
	CheckURL         []string `yaml:"checkURL"`
	CheckRspKeywords []string `yaml:"checkRspKeywords"`
	MaxConcurrentReq int      `yaml:"maxConcurrentReq"`
	CheckInterval    int      `yaml:"checkInterval"`
	MinSize          int      `yaml:"minSize"`
}
type CheckGeolocate struct {
	Enabled                 bool     `yaml:"enabled"`
	CheckURL                []string `yaml:"checkURL"`
	ExcludeKeywords         []string `yaml:"excludeKeywords"`
	IncludeKeywords         []string `yaml:"includeKeywords"`
	IncludeKeywordCondition string   `yaml:"includeKeywordCondition"`
	ExcludeKeywordCondition string   `yaml:"excludeKeywordCondition"`
	CheckInterval           int      `yaml:"checkInterval"` // 检测间隔(秒)
}
