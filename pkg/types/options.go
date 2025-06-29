package types

type Options struct {
	ConfigPath         string
	AliveDataPath      string
	Debug              bool
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
	Customs      []*Custom     `yaml:"customs"`
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

type Custom struct {
	Endpoint      string              `yaml:"endpoint"`
	Method        string              `yaml:"method"`
	Headers       map[string]string   `yaml:"headers"`
	Body          string              `yaml:"body"`
	Extract       *ProxyExtractConfig `yaml:"extract"`
	MaxSize       int                 `yaml:"maxSize"`
	ResponseType  string              `yaml:"type"`
	EnablePaging  bool                `yaml:"enablePaging"`
	CheckInterval int                 `yaml:"checkInterval"` // 检测间隔(分钟)
	QueryTimeout  int                 `yaml:"queryTimeout"`  // 请求延迟时间
}
type ProxyExtractConfig struct {
	ProxyListPath string `yaml:"path"`      // 代理列表的 JSON/XPATH 路径，如 "data.proxies"
	IPField       string `yaml:"ipField"`   // IP 字段名，如 "ip"
	PortField     string `yaml:"portField"` // Port 字段名，如 "port"
	UserField     string `yaml:"userField"` // 用户名字段名，如 "user"
	PasswordField string `yaml:"passField"` // 密码字段名，如 "password"
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
