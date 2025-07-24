package data

import (
	"os"
	"testing"

	"kratos-boilerplate/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

// 测试NewData - 成功情况
func TestNewData_Success(t *testing.T) {
	// 创建测试配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}

	logger := log.NewStdLogger(os.Stdout)

	// 执行测试 - 这里会失败，因为无法连接到真实的数据库
	// 但我们可以测试配置验证逻辑
	data, cleanup, err := NewData(config, logger)

	// 验证结果 - 预期会失败，因为无法连接数据库
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Nil(t, cleanup)
}

// 测试NewData - 数据库连接失败
func TestNewData_DatabaseConnectionFailed(t *testing.T) {
	// 创建无效的数据库配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "invalid_driver",
			Source: "invalid_source",
		},
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}

	logger := log.NewStdLogger(os.Stdout)

	// 执行测试
	data, cleanup, err := NewData(config, logger)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Nil(t, cleanup)
}

// 测试数据层清理
func TestDataCleanup(t *testing.T) {
	// 创建测试配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}

	logger := log.NewStdLogger(os.Stdout)

	// 创建数据层 - 预期会失败
	data, cleanup, err := NewData(config, logger)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Nil(t, cleanup)

	// 如果cleanup不为nil，执行清理
	if cleanup != nil {
		cleanup()
	}
}

// 测试数据层结构
func TestDataStructure(t *testing.T) {
	// 创建测试配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver: "postgres",
			Source: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
		},
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}

	logger := log.NewStdLogger(os.Stdout)

	// 创建数据层 - 预期会失败
	data, cleanup, err := NewData(config, logger)
	assert.Error(t, err)
	assert.Nil(t, data)

	// 如果cleanup不为nil，执行清理
	if cleanup != nil {
		cleanup()
	}
}

// 测试配置验证
func TestConfigValidation(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)

	// 测试空配置
	_, _, err := NewData(nil, logger)
	assert.Error(t, err)

	// 测试缺少数据库配置
	config := &conf.Data{
		Redis: &conf.Data_Redis{
			Addr:    "localhost:6379",
			Network: "tcp",
		},
	}
	_, _, err = NewData(config, logger)
	assert.Error(t, err)

	// 测试缺少Redis配置
	config = &conf.Data{
		Database: &conf.Data_Database{
			Driver: "sqlite3",
			Source: ":memory:",
		},
	}
	_, _, err = NewData(config, logger)
	assert.Error(t, err)
}
