# go-boot-viper

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-viper)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-viper)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-viper/test.yml?branch=master)](https://github.com/xudefa/go-boot-viper/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-viper.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-viper) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-viper)](https://goreportcard.com/report/github.com/xudefa/go-boot-viper)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 Viper 配置管理集成模块。将 spf13/viper 无缝集成到 go-boot 的 IoC 容器和自动配置体系中，提供配置文件加载、环境变量绑定、配置变更监听等能力。

> 设计理念：遵循 go-boot 的开发规范，通过函数式选项模式和自动配置实现零代码启动配置管理服务。

## 整体架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                    go-boot ApplicationContext                         │
│  ┌───────────┐ ┌──────────────┐ ┌───────────┐ ┌───────────┐           │
│  │ Container │ │  Environment │ │ Lifecycle │ │ EventBus  │           │
│  └───────────┘ └──────────────┘ └───────────┘ └───────────┘           │
│                       ┌─────────────────────┐                         │
│                       │ AutoConfig Registry │                         │
│                       └─────────────────────┘                         │
└───────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │    go-boot-viper Starter      │
                    │  ┌─────────────────────────┐  │
                    │  │ ViperConfig Bean        │  │
                    │  │ File Loader             │  │
                    │  │ Watch Manager           │  │
                    │  │ Environment Binding     │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [配置访问](#配置访问)
- [高级功能](#高级功能)
- [配置选项](#配置选项)
- [项目结构](#项目结构)
- [开发指南](#开发指南)
- [贡献](#贡献)
- [许可证](#许可证)

## 快速开始

### 安装

```bash
# 安装核心框架
go get github.com/xudefa/go-boot

# 安装 Viper 集成模块
go get github.com/xudefa/go-boot-viper
```

### 最小示例

```go
package main

import (
    "fmt"

    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/config"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-config-app"),
        boot.WithVersion("1.0.0"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 启动应用（自动加载配置文件）
    app.Start()

    // 获取配置实例并读取配置
    cfg := app.Container().Get("viperConfig").(config.Config)
    
    host := cfg.GetString("server.host")
    port := cfg.GetInt("server.port")
    fmt.Printf("Server: %s:%d\n", host, port)

    // 等待终止信号
    app.WaitForSignal()
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| 多格式支持 | 支持 YAML、JSON、TOML 等配置文件格式 |
| 自动配置 | 通过环境变量自动加载配置文件 |
| 环境变量绑定 | 支持环境变量覆盖配置值 |
| 配置监听 | 支持配置文件热更新和变更回调 |
| 类型安全 | 支持结构体映射和类型安全访问 |
| 加密配置 | 支持加密配置文件解密 |
| 环境区分 | 支持多环境配置文件（dev、prod 等） |

## 配置访问

### 基本访问

```go
cfg := app.Container().Get("viperConfig").(config.Config)

// 字符串类型
host := cfg.GetString("server.host")

// 整数类型
port := cfg.GetInt("server.port")

// 布尔类型
debug := cfg.GetBool("server.debug")

// 持续时间
timeout := cfg.GetDuration("server.timeout")

// 字符串切片
origins := cfg.GetStringSlice("cors.origins")
```

### 结构体映射

```go
type ServerConfig struct {
    Host    string `mapstructure:"host"`
    Port    int    `mapstructure:"port"`
    Timeout int    `mapstructure:"timeout"`
}

var serverCfg ServerConfig
err := cfg.UnmarshalKey("server", &serverCfg)
// 或
err := cfg.Unmarshal(&serverCfg)
```

### 嵌套配置访问

```go
// application.yml
// server:
//   host: localhost
//   port: 8080
//   ssl:
//     enabled: true
//     cert: /path/to/cert.pem

enabled := cfg.GetBool("server.ssl.enabled")
cert := cfg.GetString("server.ssl.cert")
```

## 高级功能

### 配置文件监听

```go
cfg := app.Container().Get("viperConfig").(*viper.ViperConfig)

cfg.Watch(context.Background(), func(event config.WatchEvent) {
    fmt.Printf("配置变更: %s = %v (旧值: %v)\n", 
        event.Key, event.NewValue, event.OldValue)
})
```

### 高级配置选项

```go
import "github.com/xudefa/go-boot-viper/viper"

// 创建高级配置实例
cfg, err := viper.NewAdvanced(
    viper.WithAdvancedConfigName("app"),
    viper.WithAdvancedConfigType("yaml"),
    viper.WithAdvancedConfigPath("./config", "/etc/myapp"),
    viper.WithAdvancedEnvironment("prod"),
    viper.WithAdvancedEnvPrefix("MYAPP"),
    viper.WithAdvancedAutoEnv(true),
    viper.WithAdvancedWatchConfig(true, func(event fsnotify.Event) {
        fmt.Println("配置文件已变更")
    }),
)
```

### 安全配置

```go
cfg, err := viper.NewAdvanced(
    viper.WithAdvancedConfigName("app"),
    viper.WithAdvancedSecureConfig(true),
    viper.WithAdvancedSecureConfigFile("config.encrypted"),
    viper.WithAdvancedSecureKey("my-secret-key"),
)
```

### 类型安全映射

```go
import "github.com/mitchellh/mapstructure"

cfg, err := viper.NewAdvanced(
    viper.WithAdvancedConfigName("app"),
    viper.WithAdvancedTypeSafe(true),
    viper.WithAdvancedDecoderConfig(&mapstructure.DecoderConfig{
        WeaklyTypedInput: true,
        Result:           &myConfig,
    }),
)
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `viper.enabled` | `true` | 是否启用 Viper 配置管理 |
| `viper.config-name` | `config` | 配置文件名 |
| `viper.config-type` | `yaml` | 配置文件类型（yaml/json/toml） |
| `viper.config-paths` | `.,./config` | 配置文件搜索路径 |
| `viper.env` | `` | 环境标识（dev/prod/test） |
| `viper.env-prefix` | `` | 环境变量前缀 |

### 示例配置

```yaml
# application.yml
viper:
  enabled: true
  config-name: app
  config-type: yaml
  config-paths: "./config,./conf"
  env: prod
  env-prefix: MYAPP
```

### 配置文件示例

```yaml
# config.yaml
server:
  host: localhost
  port: 8080
  timeout: 30

database:
  host: localhost
  port: 5432
  name: mydb
  user: admin
  password: secret

redis:
  address: localhost:6379
  db: 0
```

## 项目结构

```
go-boot-viper/
├── viper.go                # Viper 配置实现
├── options.go              # 高级选项配置
├── autoconfig.go           # 自动配置注册
├── viper_test.go           # 支持自定义刷新配置选项
├── README.md
├── LICENSE
└── go.mod
```

## 开发指南

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
go test -cover ./...       # 带覆盖率
go test -race ./...        # 数据竞争检测
```

### 代码规范

```bash
go fmt ./...
golangci-lint run
```

## 贡献

欢迎提交 Issue 和 Pull Request！详细贡献指南请参阅 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证 — 详情请参阅 [LICENSE](./LICENSE) 文件。