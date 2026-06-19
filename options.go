// Package viper 提供 Viper 配置管理的高级选项。
//
// 该包定义了 Viper 配置的高级选项，支持更精细的配置管理。
package viper

import (
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
)

// AdvancedConfigOption 高级 Viper 配置选项函数类型
type AdvancedConfigOption func(*advancedConfig)

// advancedConfig 高级配置结构
type advancedConfig struct {
	configName  string   // 配置文件名
	configType  string   // 配置文件类型（yaml、json、toml 等）
	configPaths []string // 配置文件搜索路径
	configFile  string   // 配置文件完整路径
	env         string   // 环境标识（dev、prod 等）

	envPrefix      string            // 环境变量前缀
	envKeyReplacer *strings.Replacer // 环境变量键名替换器
	autoEnv        bool              // 是否自动绑定环境变量

	watchConfig    bool                    // 是否监听配置文件变更
	onConfigChange func(in fsnotify.Event) // 配置变更回调

	secureConfig     bool   // 是否启用安全配置
	secureConfigFile string // 加密配置文件路径
	secureKey        string // 加密密钥

	typeSafe      bool                        // 是否启用类型安全
	decoderConfig *mapstructure.DecoderConfig // 类型解码器配置
}

// WithAdvancedConfigName 设置配置文件名
func WithAdvancedConfigName(name string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.configName = name
	}
}

// WithAdvancedConfigType 设置配置文件类型
func WithAdvancedConfigType(configType string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.configType = configType
	}
}

// WithAdvancedConfigPath 设置配置文件路径
func WithAdvancedConfigPath(paths ...string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.configPaths = append(c.configPaths, paths...)
	}
}

// WithAdvancedConfigFile 设置配置文件
func WithAdvancedConfigFile(file string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.configFile = file
	}
}

// WithAdvancedEnvironment 设置环境
func WithAdvancedEnvironment(env string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.env = env
	}
}

// WithAdvancedEnvPrefix 设置环境变量前缀
func WithAdvancedEnvPrefix(prefix string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.envPrefix = prefix
	}
}

// WithAdvancedEnvKeyReplacer 设置环境变量键替换器
func WithAdvancedEnvKeyReplacer(replacer *strings.Replacer) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.envKeyReplacer = replacer
	}
}

// WithAdvancedAutoEnv 启用自动环境变量
func WithAdvancedAutoEnv(enable bool) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.autoEnv = enable
	}
}

// WithAdvancedWatchConfig 启用配置文件监听
func WithAdvancedWatchConfig(enable bool, onChange func(in fsnotify.Event)) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.watchConfig = enable
		c.onConfigChange = onChange
	}
}

// WithSecureConfig 启用安全配置
func WithSecureConfig(enable bool, configFile, key string) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.secureConfig = enable
		c.secureConfigFile = configFile
		c.secureKey = key
	}
}

// WithTypeSafe 启用类型安全
func WithTypeSafe(enable bool, decoderConfig *mapstructure.DecoderConfig) AdvancedConfigOption {
	return func(c *advancedConfig) {
		c.typeSafe = enable
		c.decoderConfig = decoderConfig
	}
}
