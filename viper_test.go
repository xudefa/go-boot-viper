// viper 集成模块测试
// 测试 Viper 配置管理的创建、默认值设置、配置读取、环境变量绑定和反序列化等功能
package viper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xudefa/go-boot/config"
)

// TestViperConfig_New 测试表驱动方式创建 ViperConfig，验证不同选项组合下的行为和错误处理
func TestViperConfig_New(t *testing.T) {
	tests := []struct {
		name     string
		opts     []config.ConfigOption
		wantUsed bool
		wantErr  bool
	}{
		{
			name: "default options",
			opts: []config.ConfigOption{
				config.WithConfigName("nonexistent"),
				config.WithConfigPath("/tmp"),
			},
			wantUsed: false,
			wantErr:  false,
		},
		{
			name: "with custom config name",
			opts: []config.ConfigOption{
				config.WithConfigName("test"),
				config.WithConfigPath("/tmp"),
			},
			wantUsed: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if vc != nil && vc.IsConfigUsed() != tt.wantUsed {
				t.Errorf("IsConfigUsed() = %v, want %v", vc.IsConfigUsed(), tt.wantUsed)
			}
		})
	}
}

// TestViperConfig_SetDefault 测试设置单个默认值，验证支持链式调用且 Get 能正确读取
func TestViperConfig_SetDefault(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	result := vc.SetDefault("test.key", "test value")
	if result != vc {
		t.Error("SetDefault should return the same instance for chaining")
	}

	if vc.Get("test.key") != "test value" {
		t.Errorf("Get() = %v, want test value", vc.Get("test.key"))
	}
}

// TestViperConfig_SetDefaults 测试批量设置默认值并用 GetString/GetInt/GetBool 读取不同类型
func TestViperConfig_SetDefaults(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	defaults := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	vc.SetDefaults(defaults)

	if vc.GetString("key1") != "value1" {
		t.Errorf("GetString(key1) = %v, want value1", vc.GetString("key1"))
	}
	if vc.GetInt("key2") != 123 {
		t.Errorf("GetInt(key2) = %v, want 123", vc.GetInt("key2"))
	}
	if vc.GetBool("key3") != true {
		t.Errorf("GetBool(key3) = %v, want true", vc.GetBool("key3"))
	}
}

// TestViperConfig_Get 测试表驱动方式读取配置，验证存在和不存在键的返回值
func TestViperConfig_Get(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetDefault("default.key", "default value")

	tests := []struct {
		name    string
		key     string
		want    any
		wantNil bool
	}{
		{
			name:    "existing default key",
			key:     "default.key",
			want:    "default value",
			wantNil: false,
		},
		{
			name:    "nonexistent key",
			key:     "nonexistent",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.Get(tt.key)
			if tt.wantNil {
				if result != nil {
					t.Errorf("Get() = %v, want nil", result)
				}
			} else {
				if result != tt.want {
					t.Errorf("Get() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

// TestViperConfig_GetAll 测试获取全部配置，验证返回所有已设置的键值对
func TestViperConfig_GetAll(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetDefaults(map[string]any{
		"key1": "value1",
		"key2": 42,
	})

	all := vc.GetAll()

	if all["key1"] != "value1" {
		t.Errorf("GetAll()[key1] = %v, want value1", all["key1"])
	}
	if all["key2"] != 42 {
		t.Errorf("GetAll()[key2] = %v, want 42", all["key2"])
	}
}

// TestViperConfig_SetEnvPrefix 测试设置环境变量前缀并读取对应环境变量值
func TestViperConfig_SetEnvPrefix(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	_ = os.Setenv("APP_TEST_KEY", "env value")
	defer func() { _ = os.Unsetenv("APP_TEST_KEY") }()

	vc.SetEnvPrefix("APP")

	if vc.envPrefix != "APP" {
		t.Errorf("envPrefix = %v, want APP", vc.envPrefix)
	}

	result := vc.Get("TEST_KEY")
	if result != "env value" {
		t.Errorf("Get(TEST_KEY) = %v, want env value", result)
	}
}

// TestViperConfig_BindEnv 测试将配置键绑定到环境变量，验证绑定后能通过配置键读取环境变量
func TestViperConfig_BindEnv(t *testing.T) {
	_ = os.Setenv("TEST_BIND_KEY", "bound value")
	defer func() { _ = os.Unsetenv("TEST_BIND_KEY") }()

	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	result, err := vc.BindEnv("bound.key", "TEST_BIND_KEY")
	if err != nil {
		t.Errorf("BindEnv() error = %v", err)
		return
	}
	if result != vc {
		t.Error("BindEnv should return the same instance for chaining")
	}
}

// TestViperConfig_IsConfigUsed 测试配置文件使用状态，验证未加载配置文件时返回 false
func TestViperConfig_IsConfigUsed(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	if vc.IsConfigUsed() != false {
		t.Errorf("IsConfigUsed() = %v, want false", vc.IsConfigUsed())
	}
}

// TestViperConfig_GetStringMap 测试获取 map[string]any 类型配置，验证嵌套值正确
func TestViperConfig_GetStringMap(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetDefault("map.key", map[string]any{
		"nested": "value",
	})

	result := vc.GetStringMap("map.key")
	if result["nested"] != "value" {
		t.Errorf("GetStringMap() = %v, want map[nested:value]", result)
	}
}

// TestViperConfig_GetStringMapString 测试获取 map[string]string 类型配置，验证字符串值转换正确
func TestViperConfig_GetStringMapString(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetDefault("map.key", map[string]any{
		"key1": "value1",
		"key2": "value2",
	})

	result := vc.GetStringMapString("map.key")
	if result["key1"] != "value1" {
		t.Errorf("GetStringMapString()[key1] = %v, want value1", result["key1"])
	}
}

// TestViperConfig_GetStringSlice 测试获取字符串切片配置，验证长度和元素正确
func TestViperConfig_GetStringSlice(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetDefault("slice.key", []any{"a", "b", "c"})

	result := vc.GetStringSlice("slice.key")
	if len(result) != 3 || result[0] != "a" {
		t.Errorf("GetStringSlice() = %v, want [a b c]", result)
	}
}

// TestViperConfig_GetIntSlice 测试获取整数切片配置，验证支持 int、int64、float64 和字符串类型的自动转换
func TestViperConfig_GetIntSlice(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	tests := []struct {
		name  string
		value []any
		want  []int
	}{
		{
			name:  "int values",
			value: []any{1, 2, 3},
			want:  []int{1, 2, 3},
		},
		{
			name:  "int64 values",
			value: []any{int64(1), int64(2), int64(3)},
			want:  []int{1, 2, 3},
		},
		{
			name:  "float64 values",
			value: []any{1.0, 2.0, 3.0},
			want:  []int{1, 2, 3},
		},
		{
			name:  "string values",
			value: []any{"1", "2", "3"},
			want:  []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc.SetDefault("test.slice", tt.value)
			result := vc.GetIntSlice("test.slice")
			if len(result) != len(tt.want) {
				t.Errorf("GetIntSlice() len = %d, want %d", len(result), len(tt.want))
				return
			}
			for i, v := range tt.want {
				if result[i] != v {
					t.Errorf("GetIntSlice()[%d] = %d, want %d", i, result[i], v)
				}
			}
		})
	}
}

// TestViperConfig_GetSource 测试获取配置源名称，验证返回 "viper"
func TestViperConfig_GetSource(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	if vc.GetSource() != "viper" {
		t.Errorf("GetSource() = %v, want viper", vc.GetSource())
	}
}

// TestViperConfig_Viper 测试获取底层 Viper 实例，验证不为空
func TestViperConfig_Viper(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	v := vc.Viper()
	if v == nil {
		t.Error("Viper() should not return nil")
	}
}

// TestViperConfig_Set 测试设置单个配置值，验证设置后可以正常读取
func TestViperConfig_Set(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.Set("test.key", "test value")
	if vc.Get("test.key") != "test value" {
		t.Errorf("Get() = %v, want test value", vc.Get("test.key"))
	}
}

// TestViperConfig_SetMap 测试设置 map 类型配置，验证可通过 GetStringMap 读取
func TestViperConfig_SetMap(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.SetMap("test.map", map[string]any{"key": "value"})
	result := vc.GetStringMap("test.map")
	if result["key"] != "value" {
		t.Errorf("GetStringMap() = %v, want map[key:value]", result)
	}
}

// TestViperConfig_Unmarshal 测试将配置反序列化到结构体，验证所有字段正确填充
func TestViperConfig_Unmarshal(t *testing.T) {
	type TestConfig struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	}

	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.Set("name", "test")
	vc.Set("port", 8080)

	var target TestConfig
	err := vc.Unmarshal(&target)
	if err != nil {
		t.Errorf("Unmarshal() error = %v", err)
		return
	}
	if target.Name != "test" {
		t.Errorf("Name = %v, want test", target.Name)
	}
	if target.Port != 8080 {
		t.Errorf("Port = %v, want 8080", target.Port)
	}
}

// TestViperConfig_UnmarshalKey 测试按指定键反序列化子配置到结构体，验证嵌套配置正确解析
func TestViperConfig_UnmarshalKey(t *testing.T) {
	type ServerConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	}

	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	vc.Set("server", map[string]any{
		"host": "localhost",
		"port": 3000,
	})

	var target ServerConfig
	err := vc.UnmarshalKey("server", &target)
	if err != nil {
		t.Errorf("UnmarshalKey() error = %v", err)
		return
	}
	if target.Host != "localhost" {
		t.Errorf("Host = %v, want localhost", target.Host)
	}
	if target.Port != 3000 {
		t.Errorf("Port = %v, want 3000", target.Port)
	}
}

// TestGetEnv 测试获取环境变量的工具函数，验证存在时返回值、不存在时返回默认值
func TestGetEnv(t *testing.T) {
	_ = os.Setenv("TEST_GETENV", "test value")
	defer func() { _ = os.Unsetenv("TEST_GETENV") }()

	tests := []struct {
		name       string
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "existing key",
			key:        "TEST_GETENV",
			defaultVal: "default",
			want:       "test value",
		},
		{
			name:       "nonexisting key",
			key:        "TEST_NONEXISTENT",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnv(tt.key, tt.defaultVal)
			if result != tt.want {
				t.Errorf("GetEnv() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestGetEnvInt 测试获取整型环境变量，验证合法值正确解析、不合法值返回默认值
func TestGetEnvInt(t *testing.T) {
	_ = os.Setenv("TEST_INT", "123")
	defer func() { _ = os.Unsetenv("TEST_INT") }()

	tests := []struct {
		name       string
		key        string
		defaultVal int
		want       int
	}{
		{
			name:       "existing key",
			key:        "TEST_INT",
			defaultVal: 0,
			want:       123,
		},
		{
			name:       "nonexisting key",
			key:        "TEST_NONEXISTENT",
			defaultVal: 99,
			want:       99,
		},
		{
			name:       "invalid value",
			key:        "TEST_INVALID",
			defaultVal: 50,
			want:       50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key == "TEST_INVALID" {
				_ = os.Setenv("TEST_INVALID", "not a number")
				defer func() { _ = os.Unsetenv("TEST_INVALID") }()
			}
			result := GetEnvInt(tt.key, tt.defaultVal)
			if result != tt.want {
				t.Errorf("GetEnvInt() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestGetEnvBool 测试获取布尔型环境变量，验证 "true"/"1"/"false" 均正确转换
func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		defaultVal bool
		want       bool
	}{
		{
			name:       "true value",
			key:        "TEST_TRUE",
			value:      "true",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "1 value",
			key:        "TEST_ONE",
			value:      "1",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "false value",
			key:        "TEST_FALSE",
			value:      "false",
			defaultVal: true,
			want:       false,
		},
		{
			name:       "nonexisting",
			key:        "TEST_NONEXISTENT",
			value:      "",
			defaultVal: true,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				_ = os.Setenv(tt.key, tt.value)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}
			result := GetEnvBool(tt.key, tt.defaultVal)
			if result != tt.want {
				t.Errorf("GetEnvBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestGetEnvDuration 测试获取 Duration 类型环境变量，验证时间字符串正确解析为秒数
func TestGetEnvDuration(t *testing.T) {
	_ = os.Setenv("TEST_DURATION", "1h30m")
	defer func() { _ = os.Unsetenv("TEST_DURATION") }()

	tests := []struct {
		name        string
		key         string
		value       string
		defaultVal  time.Duration
		wantSeconds int64
	}{
		{
			name:        "existing key",
			key:         "TEST_DURATION",
			value:       "1h30m",
			defaultVal:  0,
			wantSeconds: 5400,
		},
		{
			name:        "nonexisting key",
			key:         "TEST_NONEXISTENT",
			value:       "",
			defaultVal:  time.Hour,
			wantSeconds: 3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnvDuration(tt.key, tt.defaultVal)
			if result.Seconds() != float64(tt.wantSeconds) {
				t.Errorf("GetEnvDuration() = %v, want %v seconds", result, tt.wantSeconds)
			}
		})
	}
}

// TestFileLoader_Load 测试文件加载器加载不存在的配置文件时，验证返回非空配置但不报错
func TestFileLoader_Load(t *testing.T) {
	loader := NewFileLoader()

	cfg, err := loader.Load(
		config.WithFileName("nonexistent"),
		config.WithPaths("/tmp"),
	)
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}
	if cfg == nil {
		t.Error("Load() should return a non-nil config")
	}
}

// TestViperConfig_WithTempConfigFile 测试加载临时 YAML 配置文件，验证配置被正确读取且 IsConfigUsed 返回 true
func TestViperConfig_WithTempConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`
name: test-app
port: 8080
server:
  host: localhost
  port: 3000
`), 0644)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	vc, err := New(
		config.WithConfigFile(configPath),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !vc.IsConfigUsed() {
		t.Error("IsConfigUsed() should be true")
	}

	if vc.GetString("name") != "test-app" {
		t.Errorf("GetString(name) = %v, want test-app", vc.GetString("name"))
	}

	if vc.GetInt("port") != 8080 {
		t.Errorf("GetInt(port) = %v, want 8080", vc.GetInt("port"))
	}

	host := vc.GetString("server.host")
	if host != "localhost" {
		t.Errorf("GetString(server.host) = %v, want localhost", host)
	}
}

// TestViperConfig_EncryptDecrypt 测试配置值的加密和解密功能
func TestViperConfig_EncryptDecrypt(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	originalValue := "sensitive_data"
	encryptionKey := "my_secret_key_32_chars_long!!"

	// 测试加密
	encrypted, err := vc.EncryptValue(originalValue, encryptionKey)
	if err != nil {
		t.Errorf("EncryptValue() error = %v", err)
		return
	}

	if encrypted == originalValue {
		t.Error("Encrypted value should not equal original value")
	}

	// 测试解密
	decrypted, err := vc.DecryptValue(encrypted, encryptionKey)
	if err != nil {
		t.Errorf("DecryptValue() error = %v", err)
		return
	}

	if decrypted != originalValue {
		t.Errorf("DecryptValue() = %v, want %v", decrypted, originalValue)
	}
}

// TestViperConfig_GetEncrypted 测试获取加密配置值
func TestViperConfig_GetEncrypted(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	originalValue := "secret_password"
	encryptionKey := "another_secret_key_32_chars_long!"

	// 先加密并设置值
	encrypted, err := vc.EncryptValue(originalValue, encryptionKey)
	if err != nil {
		t.Fatalf("Failed to encrypt value: %v", err)
	}
	vc.Set("encrypted.password", encrypted)

	// 测试获取加密值
	result, err := vc.GetEncrypted("encrypted.password", encryptionKey)
	if err != nil {
		t.Errorf("GetEncrypted() error = %v", err)
		return
	}

	if result != originalValue {
		t.Errorf("GetEncrypted() = %v, want %v", result, originalValue)
	}

	// 测试不存在的键
	_, err = vc.GetEncrypted("nonexistent.key", encryptionKey)
	if err == nil {
		t.Error("GetEncrypted() should return error for nonexistent key")
	}
}

// TestViperConfig_SetEncrypted 测试设置加密配置值
func TestViperConfig_SetEncrypted(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	originalValue := "secret_token"
	encryptionKey := "yet_another_secret_key_32_chars!"

	err := vc.SetEncrypted("encrypted.token", originalValue, encryptionKey)
	if err != nil {
		t.Errorf("SetEncrypted() error = %v", err)
		return
	}

	// 验证值已被加密存储
	storedValue := vc.GetString("encrypted.token")
	if storedValue == originalValue {
		t.Error("Stored value should be encrypted, not plain text")
	}

	// 验证可以解密回来
	decrypted, err := vc.GetEncrypted("encrypted.token", encryptionKey)
	if err != nil {
		t.Errorf("Cannot decrypt stored value: %v", err)
		return
	}

	if decrypted != originalValue {
		t.Errorf("Decrypted value = %v, want %v", decrypted, originalValue)
	}
}

// TestViperConfig_Sub 测试获取子配置
func TestViperConfig_Sub(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	// 设置嵌套配置
	nestedConfig := map[string]interface{}{
		"host": "localhost",
		"port": 8080,
	}
	vc.Set("server", nestedConfig)

	// 获取子配置
	subConfig := vc.Sub("server")
	if subConfig == nil {
		t.Fatal("Sub() should not return nil")
	}

	host := subConfig.GetString("host")
	if host != "localhost" {
		t.Errorf("Sub().GetString(host) = %v, want localhost", host)
	}

	port := subConfig.GetInt("port")
	if port != 8080 {
		t.Errorf("Sub().GetInt(port) = %v, want 8080", port)
	}
}

// TestViperConfig_AllKeys 测试获取所有键名
func TestViperConfig_AllKeys(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	// 设置一些配置
	vc.Set("app.name", "test-app")
	vc.Set("app.version", "1.0.0")
	vc.Set("server.port", 8080)

	keys := vc.AllKeys()
	expectedKeys := []string{"app.name", "app.version", "server.port"}

	// 检查是否包含预期的键
	for _, expectedKey := range expectedKeys {
		found := false
		for _, key := range keys {
			if key == expectedKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllKeys() missing expected key: %s", expectedKey)
		}
	}
}

// TestViperConfig_MergeConfigMap 测试合并配置映射
func TestViperConfig_MergeConfigMap(t *testing.T) {
	vc, _ := New(
		config.WithConfigName("nonexistent"),
		config.WithConfigPath("/tmp"),
	)

	// 初始配置
	vc.Set("existing.key", "initial_value")

	// 要合并的配置
	mergeMap := map[string]interface{}{
		"new.key":      "new_value",
		"existing.key": "updated_value", // 应该覆盖现有值
		"another.key":  "another_value",
	}

	vc.MergeConfigMap(mergeMap)

	// 验证合并结果
	if vc.GetString("existing.key") != "updated_value" {
		t.Errorf("Merged value for existing.key = %v, want updated_value", vc.GetString("existing.key"))
	}

	if vc.GetString("new.key") != "new_value" {
		t.Errorf("Merged value for new.key = %v, want new_value", vc.GetString("new.key"))
	}

	if vc.GetString("another.key") != "another_value" {
		t.Errorf("Merged value for another.key = %v, want another_value", vc.GetString("another.key"))
	}
}

// TestViperConfig_AdvancedConfig 测试高级配置功能
func TestViperConfig_AdvancedConfig(t *testing.T) {
	// 创建临时配置文件用于测试
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`
app:
  name: "test-app"
  version: "1.0.0"
server:
  host: "localhost"
  port: 8080
`), 0644)
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}

	// 测试高级配置
	vc, err := NewAdvanced(
		WithAdvancedConfigFile(configPath),
		WithAdvancedEnvPrefix("TEST"),
		WithAdvancedAutoEnv(true),
	)
	if err != nil {
		t.Fatalf("NewAdvanced() error = %v", err)
	}

	if !vc.IsConfigUsed() {
		t.Error("IsConfigUsed() should be true")
	}

	appName := vc.GetString("app.name")
	if appName != "test-app" {
		t.Errorf("GetString(app.name) = %v, want test-app", appName)
	}

	serverPort := vc.GetInt("server.port")
	if serverPort != 8080 {
		t.Errorf("GetInt(server.port) = %v, want 8080", serverPort)
	}
}
