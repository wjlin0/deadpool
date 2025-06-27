# 简介
deadpool是一个用于代理池管理的工具，它能够自动检测代理的有效性，并提供一个代理池供其他程序使用。

# 功能
- 自动检测代理的有效性
- 提供一个代理池供其他程序使用
- 仅内网SOCKS5，但会提供高效的代理
# 使用方法
## docker-compose
```shell
mkdir ~/.deadpool
wget https://raw.githubusercontent.com/wjlin0/deadpool/refs/heads/main/scrips/compose/docker-compose-cn.yml
docker-compose -f docker-compose-cn.yml up -d

# 查看日志出现 代理可用 即可使用
docker-compose -f docker-compose-cn.yml logs
export http_proxy=socks5://127.0.0.1:1080
curl cip.cc

```


## docker
```shell
docker run -d \
  --name deadpool \
  --restart unless-stopped \
  -p 1080:1080 \
  -v ~/.deadpool:/app \
  registry.cn-hangzhou.aliyuncs.com/wjlin0/deadpool:latest
```
## 命令行
```shell
./deadpool -c config.yaml
```

# 示例配置
```yaml
listener: # 监听配置
    ip: 127.0.0.1 # 监听地址
    port: 1080 # 监听端口
    auths:  # 认证列表，留空表示无需认证  支持多个
      - user:pass
checkSock: # SOCKS5 代理检测配置
    checkURL: # 检测代理有效性的 URL 列表 支持多个
        - https://www.baidu.com
    checkRspKeywords: # 检测响应中必须包含的关键词 支持多个 只要匹配到一项就会返回
        - '百度一下'
    maxConcurrentReq: 100 # 代理检测最大并发
    checkInterval: 8 # 超时时间（单位：秒）
    minSize: 20 # 代理池最小大小
checkGeolocate: # 地理位置检测配置
    enabled: true # 是否启用地理位置检测
    checkInterval: 30 # 地理位置检测间隔（单位：秒）
    checkURL: # 地理位置检测的 URL 列表
        - https://qifu-api.baidubce.com/ip/local/geo/v1/district
        - https://ipapi.co/json
    excludeKeywords: # 排除的关键词列表
        - HK
        - 香港
    excludeKeywordCondition: or # 排除关键词的条件（and/or）
    includeKeywords: # 包含的关键词列表
        - '"country": "CN"'
        - 中国
    includeKeywordCondition: or # 包含关键词的条件（and/or）


sourcesConfig: # 代理来源配置
    hunter: # Hunter 数据源配置
        enabled: false # 是否启用 Hunter 数据源
        apiKey: "2312ba0ebxxxxxxxxxxxxx2cc48ee" # Hunter API 密钥
        endpoint: https://hunter.qianxin.com/openApi/search # Hunter API 端点
        query: 'protocol=="socks5" && protocol.banner="No authentication"&& ip.country="CN"' # 查询条件
        maxSize: 500 # 最大查询结果数量
        queryTimeout: 60 # hunter 查询间隔（单位：分）
        checkInterval: 50 # 这个参数是通过 hunter 得到的IP 对应的每一个IP存活检测的间隔（单位：秒）
    quake: # Quake 
        enabled: false # 是否启用 Quake 数据源
        apiKey: "" # Quake API 密钥
        maxSize: 500 # 最大查询结果数量
        endpoint: https://quake.360.net/api/v3/search/quake_service # Quake API 端点
        query: 'service: socks5  AND country: "CN" AND response:"No authentication"' # 查询条件
        queryTimeout: 60 # Quake 查询间隔（单位：分）
        checkInterval: 50 # 这个参数是通过 Quake 得到的IP 对应的每一个IP存活检测的间隔（单位：秒）
    file: # 文件数据源配置
        enabled: false # 是否启用文件数据源
        path: proxies.txt # 代理文件路径
        queryTimeout: 60 # 文件数据源查询间隔（单位：分）
        checkInterval: 50 # 这个参数是通过 file 得到的IP 对应的每一个IP存活检测的间隔（单位：秒）
    checkerProxy: # CheckerProxy 数据源配置
        enabled: true # 是否启用 CheckerProxy 数据源
        endpoint: https://api.checkerproxy.net/v1/landing/archive # CheckerProxy API 端点
        queryOutTime: 60 # CheckerProxy 查询间隔（单位：分）
        checkInterval: 50 # 这个参数是通过 CheckerProxy 得到的IP 对应的每一个IP存活检测的间隔（单位：秒）
```



# 贡献
如果您有任何建议或者想要贡献代码，请随时提交Pull Request或者创建Issue。

# 许可证
deadpool遵循MIT许可证发布，请查看LICENSE文件了解更多信息。
