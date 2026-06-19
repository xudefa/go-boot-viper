# go-boot-viper 项目开发规范文档

go-boot-viper 是一个基于 [github.com/xudefa/go-boot](https://github.com/xudefa/go-boot) 的 Viper 配置管理集成模块。本模块将 spf13/viper 无缝集成到 go-boot 的 IoC 容器和自动配置体系中，遵循 go-boot 项目的开发规范。

## 1. 项目定位

### 1.1 与 go-boot 的关系

- **基础框架**：go-boot 提供核心 IoC 容器、自动配置、生命周期管理等基础设施
- **集成模块**：go-boot-viper 是 go-boot 的配置层集成，将 Viper 作为 `config.Config` 接口的实现
- **规范继承**：完全遵循 go-boot 的开发规范、命名约定、代码风格

### 1.2 核心职责

- 将 Viper 配置管理器注册为 go-boot 容器中的 Bean（Bean ID: `viperConfig`）
- 实现 `config.Config` 接口的 Viper 配置适配器
- 提供配置文件加载、环境变量绑定、配置变更监听等能力
- 提供自动配置，通过 `viper.enabled=true` 条件控制
- 支持高级配置选项（加密配置、类型安全映射等）

## 2. 项目架构

### 2.1 整体架构

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
                    │  │ (config.Config)         │  │
                    │  │ File Loader             │  │
                    │  │ Watch Manager           │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

- **基础依赖**：依赖 go-boot 核心框架（`github.com/xudefa/go-boot`）
- **配置管理**：集成 Viper（`github.com/spf13/viper`）
- **职责边界**：仅负责 Viper 配置管理集成，不包含其他业务逻辑
- **示例代码**：统一放在 `examples/` 目录，演示 Viper 集成用法

### 2.2 go-boot-viper 核心包结构

| 文件 | 说明 | 主要功能 |
|---|------|----------|
| `viper.go` | Viper 配置实现 | `ViperConfig` 实现 `config.Config` 接口 |
| `options.go` | 高级选项配置 | `AdvancedConfigOption` 高级配置选项 |
| `autoconfig.go` | 自动配置注册 | `ViperAutoConfiguration` 注册 Bean |

### 2.3 go-boot 核心包参考

go-boot-viper 依赖 go-boot 的以下核心包：

| 包 | 说明 | 接口定义 |
|---|------|----------|
| `core/` | IoC 容器（依赖注入核心） | `core.Container` |
| `boot/` | 应用启动器、自动配置注册 | `boot.AutoConfiguration`, `boot.Starter` |
| `context/` | 应用上下文（聚合容器、环境、生命周期、事件） | `context.ApplicationContext` |
| `environment/` | 环境配置管理（分层 PropertySource + Profile） | `environment.Environment` |
| `condition/` | 条件判断（OnProperty 等） | `condition.Condition` |
| `config/` | 配置管理（Config 接口 + Loader 链 + Validator） | `config.Config`, `config.Loader` |

### 2.4 接口抽象原则

go-boot-viper 遵循 go-boot 的接口抽象原则，所有集成层通过核心框架中的接口抽象定义，实现运行时互换：

- `config.Config` — 配置管理
- `core.Container` — IoC 容器
- `boot.AutoConfiguration` — 自动配置
- `boot.Starter` — 启动器生命周期

## 3. 开发规范

### 3.1 命名约定

- **包名**：小写、多个单词中间用"-"连接，除开main包，其他包名和最里层目录名保持一致
- **导出标识符**：大写驼峰（`ViperConfig`）
- **非导出标识符**：小写驼峰（`viperConfig`）
- **常量**：使用驼峰，而非全大写加下划线（`DefaultViperConfigName`）
- **测试函数**：`TestFunctionName_Condition_ExpectedBehavior`
- **错误变量**：以 `Err` 前缀（`ErrConfigNotFound`）
- **接口**：通常以 `er` 后缀（`Loader`, `Watcher`）或功能描述（`Logger`, `Config`）

### 3.2 导入规范

- 使用标准库分组 → 第三方包 → 本地包，每组之间用空白行分隔
- 禁止相对导入，使用模块路径完整导入

```go
import (
    "context"
    "fmt"
    "sync"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/viper"

    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/config"
    "github.com/xudefa/go-boot/core"
)
```

### 3.3 函数式选项模式

整个框架优先使用函数式选项模式，而非建造者模式或配置结构体：

```go
// 良好 — 基础配置选项
cfg, err := viper.New(
    config.WithConfigName("app"),
    config.WithEnvironment("dev"),
    config.WithConfigPath("./config"),
)

// 良好 — 高级配置选项
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

### 3.4 注释与文档规范

#### 3.4.1 代码注释
- 使用中文注释，保持国际化友好
- 接口、结构体需要 doc 注释，接口注释需要使用示例
- 代码实现细节较复杂的，处理步骤>=3的，都需要注释说明执行逻辑和流程
- 导出类型和函数必须有文档注释
- 注释内容应说明"为什么这样做"而不是"做了什么"

#### 3.4.2 文档注释格式
```go
// New 创建 ViperConfig 实例。
// 默认配置搜索路径为 ["./", "./config"]，默认配置文件名为 "config"，
// 默认环境为 "dev"，默认配置类型为 "yaml"。
//
// 参数:
//   - opts: 可变数量的配置选项函数，用于设置配置元数据
//
// 返回:
//   - *ViperConfig: 配置实例，如果配置文件不存在返回有效实例，configUsed 为 false
//   - error: 创建过程中的错误
//
// 示例:
//
//	cfg, err := viper.New(config.WithConfigName("myapp"))
func New(opts ...config.ConfigOption) (*ViperConfig, error) {
    // implementation
}
```

### 3.5 IoC 容器规范

- Bean 注册使用 `ctx.Register("id", core.Bean(value), core.Singleton())`
- 字段注入使用 `inject:"beanId"` 结构体标签
- 自动配置通过 `boot.RegisterAutoConfig()` 注册，使用 `condition.OnProperty()` 控制启用条件

### 3.6 错误处理

- 不忽略任何返回错误
- 使用 `fmt.Errorf` 或 `errors.New`，必要时用 `%w` 包装
- 框架层错误使用 sentinel errors
- 错误信息应清晰描述问题和可能的解决方案

### 3.7 代码风格规范

#### 3.7.1 总体原则
- **清晰优于巧妙**：代码应该易于理解和维护
- **简单优于复杂**：优先选择简单直接的实现方式
- **可读性第一**：代码首先是给人阅读的，其次才是给机器执行的

#### 3.7.2 变量声明
- 非零值使用短变量声明 `:=`
- 零值初始化使用 `var`
- 切片和映射必须初始化，不允许为 nil

#### 3.7.3 控制流
- 优先处理错误和边界条件（早期返回）
- 消除不必要的 `else`
- 复杂条件提取为命名布尔变量

#### 3.7.4 函数设计
- 函数应简短专注，单一职责
- 参数不超过 4 个，超过时使用选项结构体
- `context.Context` 总是第一个参数

### 3.8 代码组织规范

#### 3.8.1 文件内组织
- 相关声明分组：类型、构造函数、方法一起
- 顺序：包文档、导入、常量、类型、构造函数、方法、辅助函数

#### 3.8.2 包组织
- 包注释应使用完整句子描述包的功能
- 相关功能应放在同一个包中
- 避免过大包，适时拆分

### 3.9 测试规范

#### 3.9.1 测试结构
- 使用表格驱动测试（table-driven tests）
- 测试函数命名：`TestFunctionName_Condition_ExpectedBehavior`
- 为边界条件和错误路径编写测试
- 并行测试：使用 `t.Parallel()` 进行并行测试

```go
func TestViperConfig_GetString(t *testing.T) {
    tests := []struct {
        name     string
        key      string
        expected string
    }{
        {
            name:     "get existing key",
            key:      "server.host",
            expected: "localhost",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test implementation
        })
    }
}
```

#### 3.9.2 测试覆盖率
- 重要功能必须有单元测试覆盖
- 关键逻辑应达到 80% 以上覆盖率
- 边界条件和错误路径应有对应测试
- 定期检查测试覆盖率，保持较高水平

#### 3.9.3 基准测试
- 对性能敏感的函数编写基准测试
- 使用 `go test -bench=. -benchmem` 运行基准测试
- 关注内存分配和 CPU 时间
- 使用 `b.ReportAllocs()` 报告内存分配情况

### 3.10 Viper 集成规范

#### 3.10.1 配置加载
- `New` 创建基础配置实例，支持配置文件名、类型、路径等选项
- `NewAdvanced` 创建高级配置实例，支持环境变量、监听、加密等选项
- 支持 YAML、JSON、TOML 等多种配置文件格式
- 支持环境标识配置文件（如 `config.dev.yaml`、`config.prod.yaml`）

#### 3.10.2 配置访问
- 实现 `config.Config` 接口的所有方法
- 支持 `GetString`, `GetInt`, `GetBool`, `GetDuration` 等类型安全访问
- 支持 `Unmarshal` 和 `UnmarshalKey` 结构体映射
- 支持 `GetStringSlice` 等集合类型访问

#### 3.10.3 配置监听
- 支持配置文件变更监听（通过 fsnotify）
- 支持配置变更回调函数
- 支持 Watch 事件发布

#### 3.10.4 高级功能
- 支持环境变量绑定和覆盖
- 支持环境变量前缀和键名替换
- 支持加密配置文件解密
- 支持类型安全映射（通过 mapstructure）

## 4. 代码质量与工具

### 4.1 构建命令

- 构建所有包：`go build ./...`

### 4.2 测试命令

- 运行所有测试：`go test ./...`
- 运行单个测试：`go test -run <TestName> ./path/to/package`
- 带覆盖率：`go test -cover ./...`
- 数据竞争检测：`go test -race ./...`

### 4.3 Lint 与格式化

- 格式化代码：`go fmt ./...`
- 静态检查：`golangci-lint run`

## 5. 应用启动与配置

### 5.1 自动配置

- 通过 `init()` 函数注册自动配置
- 启用条件：`viper.enabled=true`
- 从 Environment 读取 Viper 配置（配置名、类型、路径、环境等）
- 自动注册 `viperConfig` Bean

### 5.2 配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `viper.enabled` | `true` | 是否启用 Viper 配置管理 |
| `viper.config-name` | `config` | 配置文件名 |
| `viper.config-type` | `yaml` | 配置文件类型（yaml/json/toml） |
| `viper.config-paths` | `.,./config` | 配置文件搜索路径 |
| `viper.env` | `` | 环境标识（dev/prod/test） |
| `viper.env-prefix` | `` | 环境变量前缀 |

### 5.3 依赖注入示例

```go
type UserService struct {
    Config config.Config `inject:"viperConfig"`
}

func (s *UserService) GetDatabaseURL() string {
    host := s.Config.GetString("database.host")
    port := s.Config.GetInt("database.port")
    name := s.Config.GetString("database.name")
    return fmt.Sprintf("%s:%d/%s", host, port, name)
}
```