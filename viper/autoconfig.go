// Package viper 提供 Viper 配置管理的自动配置。
//
// 当 viper.enabled=true 时自动启用，从 Environment 中读取 viper.config-name、viper.config-type、
// viper.config-paths、viper.env-prefix 等配置项，
// 创建并注册 Viper Config Bean 到 IoC 容器中（Bean ID: viperConfig），实现 config.Config 接口。
package viper

import (
	"strings"

	vipercore "github.com/xudefa/go-boot-viper"

	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/config"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
)

// init 注册 Viper 自动配置，由 viper.enabled=true 条件控制。
func init() {
	boot.RegisterAutoConfig(&ViperAutoConfiguration{},
		condition.OnProperty(constants.ViperEnabled, constants.ConditionTrue),
	)
}

// ViperAutoConfiguration Viper 配置管理的自动配置。
//
// 从 Environment 中读取 viper.config-name、viper.config-type、viper.config-paths 等配置项，
// 创建 Viper 配置实例并注册到 IoC 容器中，实现 config.Config 接口。
// 启用条件：viper.enabled=true
type ViperAutoConfiguration struct{}

// Configure 执行自动配置逻辑，创建 ViperConfig 并注册为 Bean。
func (v *ViperAutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	env := ctx.Environment()

	configName := env.GetString(constants.ViperConfigName, constants.DefaultViperConfigName)
	configType := env.GetString(constants.ViperConfigType, constants.DefaultViperConfigType)
	configPathsStr := env.GetString(constants.ViperConfigPaths, constants.DefaultViperConfigPaths)
	configPaths := strings.Split(configPathsStr, ",")
	envPrefix := env.GetString(constants.ViperEnvPrefix, constants.DefaultViperEnvPrefix)

	opts := []config.ConfigOption{
		config.WithConfigName(configName),
		config.WithConfigType(configType),
		config.WithConfigPath(configPaths...),
	}

	if envVal := env.GetString(constants.ViperEnv, ""); envVal != "" {
		opts = append(opts, config.WithEnvironment(envVal))
	}

	vc, err := vipercore.New(opts...)
	if err != nil {
		return err
	}

	if envPrefix != "" {
		vc.SetEnvPrefix(envPrefix)
	}

	if err := ctx.Register(constants.ViperConfigBeanID,
		core.Bean(vc),
		core.Singleton(),
	); err != nil {
		return err
	}

	return nil
}

var _ config.Config = (*vipercore.ViperConfig)(nil)
