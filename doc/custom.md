# 📡 自定义数据源配置指南 (Custom Sources)

## 📌 基础配置参数

| 参数              | 类型     | 必填 | 默认值     | 说明                                                        |
|:----------------|:-------| :--- |:--------|:----------------------------------------------------------|
| `endpoint`      | string | ✅    | -       | 目标API地址，支持： • 含`{page}`和`{pageSize}`占位符的分页URL • 路径参数或查询参数 |
| `method`        | string | ❌    | `GET`   | HTTP方法：`GET`/`POST`/`PUT`/`DELETE`                        |
| `headers`       | map    | ❌    | `{}`    | 自定义请求头，格式为 `键: 值`                                         |
| `body`          | string | ❌    | -       | 请求体内容，支持`{page}`和`{pageSize}`占位符                          |
| `maxSize`       | int    | ❌    | `100`   | 最大获取数（防止Fofa类型的，造成浪费）                                     |
| `type`          | string | ❌    | `text`  | 响应类型： • `json` - JSON格式解析 • `text` - 文本行解析                |
| `enablePaging`  | bool   | ❌    | `false` | 是否启用自动分页 启动后要设置 `{page}` 占位符                               |
| `pageSize`      | int    | ❌    | `50`    | 每页请求数量（仅`enablePaging=true`时生效）                           |
| `checkInterval` | int    | ❌    | `60`    | 数据的代理存活探测的时间间隔（秒）                                         |
| `queryTimeout`  | int    | ❌    | `60`    | 数据源探测的时间间隔（分）                                             |
| `extract`       | map | ❌    | `{}`    | 响应专用配置，详见下文                                               |
------

## 🔍 响应类型专用配置

### 1️⃣ JSON响应 (`type: json`)

```yaml
extract:
  path: "data.proxies"  # 代理列表的JSON路径（点号语法）
  ipField: "ip"         # IP字段名
  portField: "port"     # 端口字段名
  userField: "username"     # 用户名字段名 若socks5代理有这个字段可以选择配置
  passField: "password" # 密码字段名 若socks5代理有这个字段可以选择配置
```

**路径解析示例**：

```text
{
  "data": {
    "proxies": [
      { "ip": "1.1.1.1", "port": 1080, "username": "user1", "password": "pass1" },
      { "ip": "2.2.2.2", "port": 2080, "username": "user2", "password": "pass2" }
    ]
  }
}
```
如果 是这样的情况 下列 情况 ，那么 可以不用担心  `ipField`、`portField`、`userField`、`passField` 的配置就无效了。

```json
{
  "data": {
    "proxies": [
      "127.0.0.1:1080"
    ]
  }
}
```



------

### XPATH解析 (`type: xpath`)
```yaml
extract:
  path: "//div[@class='proxy-list']//table//tr"  # 代理列表的JSON路径（点号语法）
  ipField: "td[1]"         # IP字段名
  portField: "td[2]"     # 端口字段名
  userField: "td[3]"     # 用户名字段名 若socks5代理有这个字段可以选择配置
  passField: "td[4]" # 密码字段名 若socks5代理有这个字段可以选择配置
```
**路径解析示例**：
```html
<table>
    <tr>
        <td>1.1.1.1</td>
        <td>1080</td>
        <td>user1</td>
        <td>pass1</td>
    </tr>
    <tr>
        <td>2.2.2.2</td>
        <td>2080</td>
        <td>user2</td>
        <td>pass2</td>     
    </tr>
</table>
```

### 2️⃣ 文本响应 (`type: text`)

自动支持以下格式：

```text
1.1.1.1:1080
2.2.2.2:2080
socks5://3.3.3.3:3080
```

⚠️ 注意：非`socks5://`前缀的地址会自动添加协议头

------

## 🔄 分页机制详解

### 启用条件

```yaml
enablePaging: true
```

### 必须包含的占位符

| 占位符       | 说明     | 示例                   |
| :----------- | :------- | :--------------------- |
| `{page}`     | 动态页码 | `/api?page={page}`     |
| `{pageSize}` | 每页数量 | `&per_page={pageSize}` |

### 工作流程

1. 从第1页开始请求
2. 自动替换URL/body中的占位符
3. 持续请求直到返回空数据
4. 按`checkInterval`定期轮询

------

## 🛠️ 完整配置示例

### 示例1：带分页的JSON API

```yaml
customs:
- endpoint: https://proxy.scdn.io/api/proxy_list.php?page={page}&per_page=50&type=SOCKS5&country=%E4%B8%AD%E5%9B%BD
  headers:
      Pragma: no-cache
      Cache-Control: no-cache
      Sec-Ch-Ua-Platform: '"macOS"'
      User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36
      Sec-Ch-Ua: '"Google Chrome";v="137", "Chromium";v="137", "Not/A)Brand";v="24"'
      Sec-Ch-Ua-Mobile: ?0
      Accept: "*/*"
      Sec-Fetch-Site: same-origin
      Sec-Fetch-Mode: cors
      Sec-Fetch-Dest: empty
      Referer: https://proxy.scdn.io/?page=1&per_page=500&type=SOCKS5&country=%E4%B8%AD%E5%9B%BD
      Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
      Priority: u=1, i
      Connection: close
  method: GET
  enablePaging: true
  type: json
  maxSize: 500
  json:
    path: "data.proxies"
    ipField: ip
    portField: port
  checkInterval: 50
  queryTimeout: 5
```

### 示例2：文本格式API（无分页）



```yaml
customs:
  - endpoint: "http://free-proxy.com/socks5.txt"
    type: text
    queryTimeout: 5
```

------


## 🎉更多案例
### fofa
⚠️ 注意替换你的 `key`
```yaml
- endpoint: https://fofa.info/api/v1/search/all?&key=your_key&fields=ip,port&page={page}&size=50&qbase64=cHJvdG9jb2w9PSJzb2NrczUiICYmIGNvdW50cnk9IkNOIiAmJiBiYW5uZXI9Ik1ldGhvZDpObyBBdXRoZW50aWNhdGlvbiI=
  method: GET
  enablePaging: true
  type: json
  checkInterval: 50
  maxSize: 150
  queryTimeout: 5
  extract:
    path: "results" # 这里的结果是 数组 所以 ipField portField 为对应的索引值即可
    ipField: 0
    portField: 1

```
### zoomeye
⚠️ 注意替换你的 `key`
```yaml
- endpoint: https://api.zoomeye.org/v2/search
  method: POST
  enablePaging: true
  type: json
  checkInterval: 50
  maxSize: 150
  queryTimeout: 60 # 一个小时运行一次
  headers:
    Content-Type: application/json
    Accept: application/json
    API-KEY: A12xxxxxxx
  body: |
    {
      "qbase64": "c2VydmljZT0ic29ja3M1IiAmJiBjb3VudHJ5PSJDTiIgJiYgYmFubmVyPSJObyBhdXRoZW50aWNhdGlvbiI=",
      "page": {page},
      "pagesize": 50,
      "fields": "ip,port"
    }
  extract:
    path: "data"
    ipField: "ip"
    portField: "port"
```

### proxy.scdn.io
```yaml
- endpoint: https://proxy.scdn.io/api/proxy_list.php?page={page}&per_page=50&type=SOCKS5&country=%E4%B8%AD%E5%9B%BD
  headers:
      Pragma: no-cache
      Cache-Control: no-cache
      Sec-Ch-Ua-Platform: '"macOS"'
      User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36
      Sec-Ch-Ua: '"Google Chrome";v="137", "Chromium";v="137", "Not/A)Brand";v="24"'
      Sec-Ch-Ua-Mobile: ?0
      Accept: "*/*"
      Sec-Fetch-Site: same-origin
      Sec-Fetch-Mode: cors
      Sec-Fetch-Dest: empty
      Referer: https://proxy.scdn.io/?page=1&per_page=500&type=SOCKS5&country=%E4%B8%AD%E5%9B%BD
      Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
      Priority: u=1, i
      Connection: close
  method: GET
  enablePaging: true
  type: json
  maxSize: 500
  extract:
    path: "data.proxies"
    ipField: ip
    portField: port
  checkInterval: 50
  queryTimeout: 5
```
### [闪臣](https://h.shanchendaili.com/)
```yaml
- endpoint: https://sch.shanchendaili.com/flow-api.html?action=get_ip&key=xxxxxxxx&count=5&type=text
  method: GET
  type: text
  queryTimeout: 4
  checkInterval: 210
```




### www.89ip.cn （没有可用的IP）

> 测试过了全是 `http & https` 的 ,socks5 这里只是提供一个势力参考

```yaml
- endpoint: https://www.89ip.cn/index_{page}.html
  method: GET
  headers:
        Pragma: no-cache
        Cache-Control: no-cache
        Cookie: https_waf_cookie=047bcfe6-502b-4a9ad1d4c0f846e988eeed7d0be5c6165f0c; Hm_lvt_f9e56acddd5155c92b9b5499ff966848=1751207737; HMACCOUNT=5F2DC6D0646631E7; https_ydclearance=09e1543455473ab608c224b7-f8d9-4460-a251-f29b4823e52f-1751215581;
        Sec-Ch-Ua: '"Google Chrome";v="137", "Chromium";v="137", "Not/A)Brand";v="24"'
        Sec-Ch-Ua-Mobile: ?0
        Sec-Ch-Ua-Platform: "macOS"
        Upgrade-Insecure-Requests: 1
        User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36
        Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7
        Sec-Fetch-Site: same-origin
        Sec-Fetch-Mode: navigate
        Sec-Fetch-User: ?1
        Sec-Fetch-Dest: document
        Referer: https://www.89ip.cn/
        Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
        Priority: u=0, i
        Connection: close
  enablePaging: true
  type: xpath
  extract:
    path: /html//tbody/tr
    ipField: td[1]
    portField: td[2]

```