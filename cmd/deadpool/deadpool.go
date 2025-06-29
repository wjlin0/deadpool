package main

import (
	"context"
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/projectdiscovery/gologger"
	"github.com/wjlin0/deadpool/pkg/runner"
	"net"
	"strings"
)

func main() {
	// 解析配置
	cfgOptions, err := runner.ParserConfigOptions(runner.ParserOptions())
	if err != nil {
		gologger.Fatal().Msg(err.Error())
		return
	}

	// 初始化代理管理器
	scpm, err := runner.NewSocksProxyManagerWithFile(cfgOptions, cfgOptions.Options.AliveDataPath)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
		return
	}

	// 启动自动维护服务
	dialFunc := scpm.Start()

	//socks.ListenSocksDefault("tcp", ":12345")
	// 创建SOCKS5服务器配置
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialFunc(network, addr)
		},
		// 如果需要认证，可以在这里配置

	}
	// 配置认证（如果设置了认证信息）
	if len(cfgOptions.Listener.Auths) > 0 {
		creds := socks5.StaticCredentials{}
		for _, auth := range cfgOptions.Listener.Auths {
			parts := strings.Split(auth, ":")
			if len(parts) == 2 {
				username := strings.TrimSpace(parts[0])
				password := strings.TrimSpace(parts[1])
				if username != "" && password != "" {
					creds[username] = password
				}
			}
		}

		// 只有有效的认证信息才设置
		if len(creds) > 0 {
			conf.AuthMethods = []socks5.Authenticator{socks5.UserPassAuthenticator{
				Credentials: creds,
			}}
			gologger.Info().Msgf("Enabled SOCKS5 authentication with %d credentials", len(creds))
		} else {
			gologger.Warning().Msg("No valid credentials found, running without authentication")
		}
	} else {
		gologger.Info().Msg("Running SOCKS5 server without authentication")
	}

	// 创建SOCKS5服务器
	server, err := socks5.New(conf)
	if err != nil {
		gologger.Fatal().Msgf("Failed to create SOCKS5 server: %v", err)
		return
	}

	// 启动监听
	listenAddr := fmt.Sprintf("%s:%d", cfgOptions.Listener.IP, cfgOptions.Listener.Port)
	gologger.Info().Msgf("Starting SOCKS5 server on %s", listenAddr)

	if err := server.ListenAndServe("tcp", listenAddr); err != nil {
		gologger.Fatal().Msgf("Failed to start SOCKS5 server: %v", err)
	}
}
