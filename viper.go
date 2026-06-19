// Package viper 基于 spf13/viper 提供配置管理实现。
//
// 该包将 viper 配置库与 go-boot 配置接口集成，
// 支持配置文件加载、环境变量、配置变更监听等功能。
//
// 定义:
//
//   - ViperConfig: 配置管理器实现了 config.Config 接口
//   - FileLoader: 配置文件加载器实现了 config.Loader 接口
//   - GetEnv/GetEnvInt/GetEnvBool/GetEnvDuration: 环境变量辅助函数
//
// 快速开始:
//
//	cfg, err := viper.New(
//	    config.WithConfigName("app"),
//	    config.WithEnvironment("dev"),
//	    config.WithConfigPath("./config"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	host := cfg.GetString("server.host")
package viper

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/xudefa/go-boot/config"
)

// ViperConfig 配置管理器.
//
// 基于 spf13/viper 实现的配置加载和管理器,
// 支持 YAML、JSON、TOML 等配置文件格式,以及环境变量覆盖.
//
// ViperConfig 通过嵌入 config.ConfigModel 继承其配置元数据字段,
// 同时实现了 config.Config 接口,提供统一的配置访问方式.
//
// Example:
//
//	cfg, err := viper.New(
//	    config.WithConfigName("app"),
//	    config.WithEnvironment("dev"),
//	    config.WithConfigPath("./config"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	host := cfg.GetString("server.host")
type ViperConfig struct {
	*config.ConfigModel
	v          *viper.Viper
	defaults   map[string]any
	envPrefix  string
	configUsed bool
	watcher    *fsnotify.Watcher
	watchMu    sync.RWMutex
	watchers   []func(config.WatchEvent)

	// 新增字段
	logger            Logger                                   // 日志记录器
	onChangeCallbacks []func(string, interface{}, interface{}) // 配置变更回调
}

// Logger 日志记录器接口，兼容多种日志库
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

// New 创建 ViperConfig 实例.
//
// 默认配置搜索路径为 ["./", "./config"],默认配置文件名为 "config",
// 默认环境为 "dev",默认配置类型为 "yaml".
//
// 参数:
//   - opts: 可变数量的配置选项函数,用于设置配置元数据
//
// 返回:
//   - *ViperConfig: 配置实例,如果配置文件不存在返回有效实例,configUsed 为 false
//   - error: 创建过程中的错误
//
// Example:
//
//	cfg, err := viper.New(config.WithConfigName("myapp"))
func New(opts ...config.ConfigOption) (*ViperConfig, error) {
	cfg := &config.ConfigModel{
		ConfigPaths: []string{"./", "./config"},
		ConfigName:  "config",
		ConfigType:  "yaml",
		Env:         "",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	vc := &ViperConfig{
		ConfigModel: cfg,
		v:           viper.New(),
		defaults:    make(map[string]any),
		envPrefix:   "",
	}

	if cfg.ConfigFile != "" {
		vc.v.SetConfigFile(cfg.ConfigFile)
	} else if cfg.ConfigName != "" {
		vc.v.SetConfigType(cfg.ConfigType)
		if cfg.Env != "" {
			vc.v.SetConfigName(cfg.ConfigName + "." + cfg.Env)
		} else {
			vc.v.SetConfigName(cfg.ConfigName)
		}
		for _, p := range cfg.ConfigPaths {
			vc.v.AddConfigPath(p)
		}
	}

	if err := vc.v.ReadInConfig(); err != nil {
		vc.configUsed = false
		return vc, nil
	}
	vc.configUsed = true
	return vc, nil
}

// NewAdvanced 创建带有高级选项的 ViperConfig 实例
//
// 支持更精细的配置管理，包括环境变量配置、配置文件监听等。
//
// 参数:
//   - opts: 可变数量的高级配置选项函数
//
// 返回:
//   - *ViperConfig: 配置实例
//   - error: 创建过程中的错误
func NewAdvanced(opts ...AdvancedConfigOption) (*ViperConfig, error) {
	advancedCfg := &advancedConfig{
		configPaths:  []string{"./", "./config"},
		configName:   "config",
		configType:   "yaml",
		env:          "",
		autoEnv:      false,
		watchConfig:  false,
		secureConfig: false,
		typeSafe:     false,
	}

	for _, opt := range opts {
		opt(advancedCfg)
	}

	cfg := &config.ConfigModel{
		ConfigPaths: advancedCfg.configPaths,
		ConfigName:  advancedCfg.configName,
		ConfigType:  advancedCfg.configType,
		Env:         advancedCfg.env,
	}

	vc := &ViperConfig{
		ConfigModel: cfg,
		v:           viper.New(),
		defaults:    make(map[string]any),
		envPrefix:   advancedCfg.envPrefix,
	}

	if advancedCfg.configFile != "" {
		vc.v.SetConfigFile(advancedCfg.configFile)
	} else if advancedCfg.configName != "" {
		vc.v.SetConfigType(advancedCfg.configType)
		if advancedCfg.env != "" {
			vc.v.SetConfigName(advancedCfg.configName + "." + advancedCfg.env)
		} else {
			vc.v.SetConfigName(advancedCfg.configName)
		}
		for _, p := range advancedCfg.configPaths {
			vc.v.AddConfigPath(p)
		}
	}

	if advancedCfg.envPrefix != "" {
		vc.v.SetEnvPrefix(advancedCfg.envPrefix)
	}

	if advancedCfg.envKeyReplacer != nil {
		vc.v.SetEnvKeyReplacer(advancedCfg.envKeyReplacer)
	}

	if advancedCfg.autoEnv {
		vc.v.AutomaticEnv()
	}

	if err := vc.v.ReadInConfig(); err != nil {
		vc.configUsed = false
		return vc, nil
	}
	vc.configUsed = true

	if advancedCfg.watchConfig && advancedCfg.onConfigChange != nil {
		vc.v.OnConfigChange(advancedCfg.onConfigChange)
		vc.v.WatchConfig()
	}

	return vc, nil
}

// MustNew 创建 ViperConfig 实例,如果出错则 panic.
//
// 当创建配置失败时,会触发 panic,这适用于确信配置必然成功的场景.
//
// 参数:
//   - opts: 可变数量的配置选项函数
//
// 返回:
//   - *ViperConfig: 配置实例
func MustNew(opts ...config.ConfigOption) *ViperConfig {
	v, err := New(opts...)
	if err != nil {
		panic(fmt.Errorf("failed to create ViperConfig: %w", err))
	}
	return v
}

// NewWithContext 使用 context 创建 ViperConfig 实例.
//
// 支持通过 context 控制配置加载超时或取消.
// 如果 context 在配置加载过程中被取消,返回 ctx.Err().
//
// 参数:
//   - ctx: 上下文,用于控制超时和取消
//   - opts: 可变数量的配置选项函数
//
// 返回:
//   - *ViperConfig: 配置实例
//   - error: 创建错误或 context 取消错误
func NewWithContext(ctx context.Context, opts ...config.ConfigOption) (*ViperConfig, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := &config.ConfigModel{
		ConfigPaths: []string{"./", "./config"},
		ConfigName:  "config",
		ConfigType:  "yaml",
		Env:         "",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	vc := &ViperConfig{
		ConfigModel: cfg,
		v:           viper.New(),
		defaults:    make(map[string]any),
		envPrefix:   "",
	}

	if cfg.ConfigFile != "" {
		vc.v.SetConfigFile(cfg.ConfigFile)
	} else if cfg.ConfigName != "" {
		vc.v.SetConfigType(cfg.ConfigType)
		if cfg.Env != "" {
			vc.v.SetConfigName(cfg.ConfigName + "." + cfg.Env)
		} else {
			vc.v.SetConfigName(cfg.ConfigName)
		}
		for _, p := range cfg.ConfigPaths {
			vc.v.AddConfigPath(p)
		}
	}

	errCh := make(chan error, 1)
	go func() {
		if err := vc.v.ReadInConfig(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			vc.configUsed = false
			return vc, nil
		}
		vc.configUsed = true
	case <-ctx.Done():
		return vc, ctx.Err()
	}

	return vc, nil
}

// SetDefault 设置配置默认值.
//
// 当配置文件中不存在该键时,使用默认值.
// 支持链式调用.
//
// 参数:
//   - key: 配置键名
//   - value: 默认值
//
// 返回:
//   - *ViperConfig: 配置实例(支持链式调用)
func (vc *ViperConfig) SetDefault(key string, value any) *ViperConfig {
	vc.defaults[key] = value
	vc.v.SetDefault(key, value)

	if vc.logger != nil {
		vc.logger.Info("default configuration value set", "key", key, "value", value)
	}

	return vc
}

// SetDefaults 批量设置配置默认值.
//
// 参数:
//   - defaults: 配置键值对
//
// 返回:
//   - *ViperConfig: 配置实例(支持链式调用)
func (vc *ViperConfig) SetDefaults(defaults map[string]any) *ViperConfig {
	for k, v := range defaults {
		vc.SetDefault(k, v)
	}
	return vc
}

// SetEnvPrefix 设置环境变量前缀.
//
// 设置后可以通过环境变量覆盖配置值.
// 自动启用 AutomaticEnv 监听环境变量.
//
// 参数:
//   - prefix: 环境变量前缀,如 "APP"
//
// 返回:
//   - *ViperConfig: 配置实例(支持链式调用)
func (vc *ViperConfig) SetEnvPrefix(prefix string) *ViperConfig {
	vc.envPrefix = prefix
	vc.v.SetEnvPrefix(prefix)
	vc.v.AutomaticEnv()
	return vc
}

// BindEnv 绑定配置键到环境变量.
//
// 将配置键与环境变量关联,使得环境变量可以覆盖配置文件值.
//
// 参数:
//   - key: 配置键名
//   - envVar: 环境变量名
//
// 返回:
//   - *ViperConfig: 配置实例(支持链式调用)
//   - error: 绑定错误
func (vc *ViperConfig) BindEnv(key string, envVar string) (*ViperConfig, error) {
	err := vc.v.BindEnv(key, envVar)
	if err != nil {
		return nil, err
	}
	return vc, nil
}

// IsConfigUsed 检查是否成功加载了配置文件.
//
// 返回:
//   - bool: 是否加载了配置文件
func (vc *ViperConfig) IsConfigUsed() bool {
	return vc.configUsed
}

// Get 根据键名获取配置值.
//
// 优先从配置文件获取,如果不存在则从默认值获取.
//
// 参数:
//   - key: 配置键名,支持点分隔的层级键名
//
// 返回:
//   - any: 配置值,如果不存在返回nil
func (vc *ViperConfig) Get(key string) any {
	if val := vc.v.Get(key); val != nil {
		if vc.logger != nil {
			vc.logger.Debug("configuration retrieved", "key", key, "value", val)
		}
		return val
	}
	if val, ok := vc.defaults[key]; ok {
		if vc.logger != nil {
			vc.logger.Debug("default configuration retrieved", "key", key, "value", val)
		}
		return val
	}
	if vc.logger != nil {
		vc.logger.Warn("configuration key not found", "key", key)
	}
	return nil
}

// GetAll 获取所有配置值,包括默认值.
//
// 返回:
//   - map[string]any: 所有配置键值对（防御性拷贝）
func (vc *ViperConfig) GetAll() map[string]any {
	settings := vc.v.AllSettings()
	for k, v := range vc.defaults {
		if _, exists := settings[k]; !exists {
			settings[k] = v
		}
	}
	// 返回防御性拷贝，防止外部修改内部状态
	result := make(map[string]any, len(settings))
	for k, v := range settings {
		result[k] = v
	}
	return result
}

// GetString 获取字符串类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - string: 字符串配置值
func (vc *ViperConfig) GetString(key string) string {
	return vc.v.GetString(key)
}

// GetStringMap 获取 map[string]any 类型的配置值，返回防御性拷贝。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - map[string]any: 配置值映射
func (vc *ViperConfig) GetStringMap(key string) map[string]any {
	m := vc.v.GetStringMap(key)
	// 返回防御性拷贝，防止外部修改内部状态
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetStringMapString 获取 map[string]string 类型的配置值，返回防御性拷贝。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - map[string]string: 配置值映射
func (vc *ViperConfig) GetStringMapString(key string) map[string]string {
	m := vc.v.GetStringMapString(key)
	// 返回防御性拷贝，防止外部修改内部状态
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// GetStringSlice 获取字符串切片类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - []string: 字符串切片配置值
func (vc *ViperConfig) GetStringSlice(key string) []string {
	return vc.v.GetStringSlice(key)
}

// GetInt 获取整数类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - int: 整数值
func (vc *ViperConfig) GetInt(key string) int {
	return vc.v.GetInt(key)
}

// GetInt64 获取 64 位整数类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - int64: 64 位整数值
func (vc *ViperConfig) GetInt64(key string) int64 {
	return vc.v.GetInt64(key)
}

// GetIntSlice 获取整数切片类型的配置值。
//
// 支持从 []any、[]int64、[]float64、[]string 自动转换。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - []int: 整数切片配置值，不存在时返回 nil
func (vc *ViperConfig) GetIntSlice(key string) []int {
	if slice := vc.v.Get(key); slice != nil {
		if arr, ok := slice.([]any); ok {
			result := make([]int, 0, len(arr))
			for _, v := range arr {
				switch val := v.(type) {
				case int:
					result = append(result, val)
				case int64:
					result = append(result, int(val))
				case float64:
					result = append(result, int(val))
				case string:
					if intVal, err := strconv.Atoi(val); err == nil {
						result = append(result, intVal)
					}
				}
			}
			return result
		}
	}
	return nil
}

// GetFloat64 获取浮点数类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - float64: 浮点数值
func (vc *ViperConfig) GetFloat64(key string) float64 {
	return vc.v.GetFloat64(key)
}

// GetBool 获取布尔类型的配置值。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - bool: 布尔值
func (vc *ViperConfig) GetBool(key string) bool {
	return vc.v.GetBool(key)
}

// HasKey 检查配置中是否存在指定的键。
//
// 参数:
//   - key: 配置键名，支持点分隔的层级键名
//
// 返回:
//   - bool: 键是否存在
func (vc *ViperConfig) HasKey(key string) bool {
	return vc.v.IsSet(key)
}

// Unmarshal 将配置映射到目标结构体.
//
// 参数:
//   - target: 目标结构体指针
//
// 返回:
//   - error: 映射错误
func (vc *ViperConfig) Unmarshal(target any) error {
	if vc.logger != nil {
		vc.logger.Debug("starting unmarshal configuration")
	}

	err := vc.v.Unmarshal(target)
	if err != nil {
		if vc.logger != nil {
			vc.logger.Error(err, "failed to unmarshal configuration")
		}
		return err
	}

	if vc.logger != nil {
		vc.logger.Debug("configuration unmarshaled successfully")
	}

	return nil
}

// UnmarshalKey 将指定键下的配置映射到目标结构体.
//
// 参数:
//   - key: 配置键名
//   - target: 目标结构体指针
//
// 返回:
//   - error: 映射错误
func (vc *ViperConfig) UnmarshalKey(key string, target any) error {
	if vc.logger != nil {
		vc.logger.Debug("starting unmarshal configuration key", "key", key)
	}

	err := vc.v.UnmarshalKey(key, target)
	if err != nil {
		if vc.logger != nil {
			vc.logger.Error(err, "failed to unmarshal configuration key", "key", key)
		}
		return err
	}

	if vc.logger != nil {
		vc.logger.Debug("configuration key unmarshaled successfully", "key", key)
	}

	return nil
}

// Watch 注册配置变更监听器.
//
// 使用 fsnotify 监听配置文件变化.
// 注意: OnConfigChange 在 viper 1.21 中已弃用但仍可用.
// 可以注册多个回调函数.
//
// 参数:
//   - callback: 配置变更回调函数
//
// 返回:
//   - error: 注册错误
func (vc *ViperConfig) Watch(callback func(config.WatchEvent)) error {
	vc.watchMu.Lock()
	defer vc.watchMu.Unlock()

	if vc.watcher == nil {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		vc.watcher = w
	}

	vc.watchers = append(vc.watchers, callback)

	vc.v.OnConfigChange(func(in fsnotify.Event) {
		event := config.WatchEvent{
			Type: config.EventModify,
			Key:  in.Name,
		}
		vc.watchMu.RLock()
		for _, cb := range vc.watchers {
			cb(event)
		}
		vc.watchMu.RUnlock()
	})

	return nil
}

// StopWatch 停止配置变更监听.
//
// 返回:
//   - error: 停止错误
func (vc *ViperConfig) StopWatch() error {
	vc.watchMu.Lock()
	defer vc.watchMu.Unlock()
	vc.watchers = nil
	if vc.watcher != nil {
		err := vc.watcher.Close()
		if err != nil {
			return err
		}
		vc.watcher = nil
	}
	return nil
}

// GetSource 获取配置源类型.
//
// 返回:
//   - string: 配置源类型,恒返回 "viper"
func (vc *ViperConfig) GetSource() string {
	return "viper"
}

// OnConfigChange 注册配置变更回调函数
//
// 当配置值发生改变时调用回调函数，传入键名、旧值和新值
//
// 参数:
//   - callback: 配置变更回调函数 func(key string, oldValue, newValue interface{})
func (vc *ViperConfig) OnConfigChange(callback func(string, interface{}, interface{})) {
	vc.onChangeCallbacks = append(vc.onChangeCallbacks, callback)
}

// AddRemoteProvider 添加远程配置提供者
//
// 支持 etcd、consul 等远程配置中心
//
// 参数:
//   - provider: 提供者名称 (如 "etcd", "consul")
//   - endpoint: 远程服务地址
//   - path: 配置路径
func (vc *ViperConfig) AddRemoteProvider(provider, endpoint, path string) error {
	return vc.v.AddRemoteProvider(provider, endpoint, path)
}

// MergeConfig 合并配置文件
//
// 将指定配置文件的内容合并到当前配置中
//
// 参数:
//   - in io.Reader: 配置输入流
func (vc *ViperConfig) MergeConfig(in io.Reader) error {
	return vc.v.MergeConfig(in)
}

// MergeConfigMap 合并配置映射
//
// 将指定映射的内容合并到当前配置中
//
// 参数:
//   - cfg: 配置映射
func (vc *ViperConfig) MergeConfigMap(cfg map[string]interface{}) {
	// 由于 viper.MergeConfigMap 不会覆盖已存在的键，我们需要手动处理
	for k, v := range cfg {
		vc.v.Set(k, v)
	}
}

// Sub 返回配置的子树
//
// 用于访问嵌套配置结构
//
// 参数:
//   - key: 子配置键名
//
// 返回:
//   - *ViperConfig: 子配置实例
func (vc *ViperConfig) Sub(key string) *ViperConfig {
	subViper := vc.v.Sub(key)
	if subViper == nil {
		return nil
	}

	subConfig := &ViperConfig{
		ConfigModel:       &config.ConfigModel{},
		v:                 subViper,
		defaults:          make(map[string]any),
		envPrefix:         vc.envPrefix,
		configUsed:        vc.configUsed,
		logger:            vc.logger,
		onChangeCallbacks: vc.onChangeCallbacks,
	}

	return subConfig
}

// AllKeys 获取所有配置键名
//
// 返回:
//   - []string: 所有键名列表
func (vc *ViperConfig) AllKeys() []string {
	return vc.v.AllKeys()
}

// AllSettings 获取所有设置（带日志）
//
// 返回:
//   - map[string]interface{}: 所有配置设置
func (vc *ViperConfig) AllSettings() map[string]interface{} {
	settings := vc.v.AllSettings()

	if vc.logger != nil {
		vc.logger.Debug("retrieved all settings", "count", len(settings))
	}

	return settings
}

// IsSet 检查配置键是否存在
//
// 参数:
//   - key: 配置键名
//
// 返回:
//   - bool: 键是否存在
func (vc *ViperConfig) IsSet(key string) bool {
	return vc.v.IsSet(key)
}

// Viper 获取底层 viper 实例.
//
// 返回:
//   - *viper.Viper: viper 实例,用于高级操作
func (vc *ViperConfig) Viper() *viper.Viper {
	return vc.v
}

// Set 设置配置值。
//
// 参数:
//   - key: 配置键名
//   - value: 配置值
func (vc *ViperConfig) Set(key string, value any) {
	oldValue := vc.v.Get(key)
	vc.v.Set(key, value)

	if vc.logger != nil {
		vc.logger.Info("configuration value set", "key", key, "old_value", oldValue, "new_value", value)
	}

	// 调用配置变更回调
	for _, callback := range vc.onChangeCallbacks {
		callback(key, oldValue, value)
	}
}

// SetMap 设置 map 类型的配置值。
//
// 参数:
//   - key: 配置键名
//   - value: map 类型配置值
func (vc *ViperConfig) SetMap(key string, value map[string]any) {
	vc.v.Set(key, value)
}

// Load 从文件加载配置。
//
// 参数:
//   - path: 配置文件路径
//
// 返回:
//   - error: 加载错误
func (vc *ViperConfig) Load(path string) error {
	vc.v.SetConfigFile(path)
	return vc.v.ReadInConfig()
}

// Save 保存配置到文件。
//
// 参数:
//   - path: 保存路径
//
// 返回:
//   - error: 保存错误
func (vc *ViperConfig) Save(path string) error {
	return vc.v.WriteConfigAs(path)
}

// EncryptValue 加密配置值
//
// 使用 AES-GCM 算法加密配置值
//
// 参数:
//   - value: 待加密的值
//   - key: 加密密钥
//
// 返回:
//   - string: 加密后的 base64 编码字符串
//   - error: 加密错误
func (vc *ViperConfig) EncryptValue(value string, key string) (string, error) {
	if len(key) < 32 {
		// 如果密钥长度不足32字节，则填充或截断
		keyBytes := []byte(key)
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		if len(keyBytes) < 32 {
			// 用零填充
			for i := len(keyBytes); i < 32; i++ {
				newKey[i] = 0
			}
		}
		key = string(newKey[:32])
	}

	block, err := aes.NewCipher([]byte(key)[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptValue 解密配置值
//
// 使用 AES-GCM 算法解密配置值
//
// 参数:
//   - encryptedValue: 加密的 base64 编码字符串
//   - key: 解密密钥
//
// 返回:
//   - string: 解密后的值
//   - error: 解密错误
func (vc *ViperConfig) DecryptValue(encryptedValue string, key string) (string, error) {
	if len(key) < 32 {
		// 如果密钥长度不足32字节，则填充或截断
		keyBytes := []byte(key)
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		if len(keyBytes) < 32 {
			// 用零填充
			for i := len(keyBytes); i < 32; i++ {
				newKey[i] = 0
			}
		}
		key = string(newKey[:32])
	}

	data, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key)[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GetEncrypted 获取加密的配置值并自动解密
//
// 参数:
//   - key: 配置键名
//   - encryptionKey: 解密密钥
//
// 返回:
//   - string: 解密后的配置值
//   - error: 获取或解密错误
func (vc *ViperConfig) GetEncrypted(key, encryptionKey string) (string, error) {
	encryptedValue := vc.v.GetString(key)
	if encryptedValue == "" {
		return "", fmt.Errorf("configuration key '%s' not found or empty", key)
	}

	return vc.DecryptValue(encryptedValue, encryptionKey)
}

// SetEncrypted 设置加密的配置值
//
// 参数:
//   - key: 配置键名
//   - value: 待加密的值
//   - encryptionKey: 加密密钥
//
// 返回:
//   - error: 加密或设置错误
func (vc *ViperConfig) SetEncrypted(key, value, encryptionKey string) error {
	encryptedValue, err := vc.EncryptValue(value, encryptionKey)
	if err != nil {
		return err
	}

	vc.v.Set(key, encryptedValue)
	return nil
}

// FileLoader 配置文件加载器.
//
// 实现 config.Loader 接口,用于从文件系统加载配置.
// 使用 Viper 作为底层配置解析器.
type FileLoader struct {
	paths    []string
	name     string
	fileType string
	env      string
}

// NewFileLoader 创建新的 FileLoader 实例.
//
// 默认搜索路径: ["./", "./config"]
// 默认文件名: "config"
// 默认类型: "yaml"
// 默认环境: "dev"
func NewFileLoader() *FileLoader {
	return &FileLoader{
		paths:    []string{"./", "./config"},
		name:     "config",
		fileType: "yaml",
		env:      "",
	}
}

// Priority 获取加载器优先级.
//
// 返回:
//   - int: 优先级,越高越先被调用
func (f *FileLoader) Priority() int {
	return 30
}

// Name 获取加载器名称.
//
// 返回:
//   - string: 加载器名称
func (f *FileLoader) Name() string {
	return "viper-file"
}

// Load 加载配置.
//
// 参数:
//   - opts: 加载器选项
//
// 返回:
//   - config.Config: 配置实例
//   - error: 加载错误
func (f *FileLoader) Load(opts ...config.LoaderOption) (config.Config, error) {
	model := &config.LoaderModel{
		Paths:    f.paths,
		FileName: f.name,
		FileType: f.fileType,
		Env:      f.env,
	}
	for _, opt := range opts {
		if err := opt(model); err != nil {
			return nil, err
		}
	}

	vc, err := New(
		config.WithConfigPath(model.Paths...),
		config.WithConfigName(model.FileName),
		config.WithConfigType(model.FileType),
		config.WithEnvironment(model.Env),
	)
	if err != nil {
		return nil, err
	}

	return vc, nil
}

// SupportsWatch 返回是否支持配置变更监听。
//
// 返回:
//   - bool: 是否支持监听（始终返回 true）
func (f *FileLoader) SupportsWatch() bool {
	return true
}

// GetEnv 获取环境变量值(字符串类型).
//
// 参数:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回:
//   - string: 环境变量值,如果不存在返回默认值
func GetEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取环境变量值(整数类型).
//
// 参数:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回:
//   - int: 环境变量值,如果不存在或转换失败返回默认值
func GetEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvBool 获取环境变量值(布尔类型).
//
// 支持 "1", "t", "T", "true", "TRUE", "0", "f", "F", "false", "FALSE".
//
// 参数:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回:
//   - bool: 环境变量值,如果不存在或转换失败返回默认值
func GetEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// GetEnvDuration 获取环境变量值(Duration类型).
//
// 参数:
//   - key: 环境变量名
//   - defaultValue: 默认值
//
// 返回:
//   - time.Duration: 环境变量值,如果不存在或转换失败返回默认值
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
