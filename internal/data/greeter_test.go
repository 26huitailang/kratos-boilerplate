package data

import (
	"context"
	"os"
	"testing"

	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

// 测试NewGreeterRepo
func TestNewGreeterRepo(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	assert.NotNil(t, repo)
}

// 测试GreeterRepo接口实现
func TestGreeterRepoInterface(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)

	// 验证实现了接口
	var _ biz.GreeterRepo = repo
	assert.NotNil(t, repo)
}

// 测试Save方法
func TestGreeterRepo_Save(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 测试数据
	greeter := &biz.Greeter{
		Hello: "test hello",
	}

	// 执行测试
	ctx := context.Background()
	result, err := greeterRepo.Save(ctx, greeter)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, greeter.Hello, result.Hello)
}

// 测试Update方法
func TestGreeterRepo_Update(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 测试数据
	greeter := &biz.Greeter{
		Hello: "updated hello",
	}

	// 执行测试
	ctx := context.Background()
	result, err := greeterRepo.Update(ctx, greeter)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, greeter.Hello, result.Hello)
}

// 测试FindByID方法
func TestGreeterRepo_FindByID(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 执行测试
	ctx := context.Background()
	result, err := greeterRepo.FindByID(ctx, 1)

	// 验证结果
	assert.NoError(t, err)
	assert.Nil(t, result) // 当前实现返回nil
}

// 测试ListByHello方法
func TestGreeterRepo_ListByHello(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 执行测试
	ctx := context.Background()
	result, err := greeterRepo.ListByHello(ctx, "test")

	// 验证结果
	assert.NoError(t, err)
	assert.Nil(t, result) // 当前实现返回nil
}

// 测试ListAll方法
func TestGreeterRepo_ListAll(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 执行测试
	ctx := context.Background()
	result, err := greeterRepo.ListAll(ctx)

	// 验证结果
	assert.NoError(t, err)
	assert.Nil(t, result) // 当前实现返回nil
}

// 测试GreeterRepo结构
func TestGreeterRepoStructure(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	// 验证结构
	assert.NotNil(t, greeterRepo.data)
	assert.NotNil(t, greeterRepo.log)
}

// 测试Greeter数据结构
func TestGreeterStructure(t *testing.T) {
	// 创建Greeter实例
	greeter := &biz.Greeter{
		Hello: "test hello",
	}

	// 验证结构
	assert.Equal(t, "test hello", greeter.Hello)
}

// 测试所有GreeterRepo方法
func TestGreeterRepoAllMethods(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	ctx := context.Background()

	// 测试Save
	greeter1 := &biz.Greeter{Hello: "hello1"}
	result1, err1 := greeterRepo.Save(ctx, greeter1)
	assert.NoError(t, err1)
	assert.Equal(t, greeter1, result1)

	// 测试Update
	greeter2 := &biz.Greeter{Hello: "hello2"}
	result2, err2 := greeterRepo.Update(ctx, greeter2)
	assert.NoError(t, err2)
	assert.Equal(t, greeter2, result2)

	// 测试FindByID
	result3, err3 := greeterRepo.FindByID(ctx, 1)
	assert.NoError(t, err3)
	assert.Nil(t, result3)

	// 测试ListByHello
	result4, err4 := greeterRepo.ListByHello(ctx, "test")
	assert.NoError(t, err4)
	assert.Nil(t, result4)

	// 测试ListAll
	result5, err5 := greeterRepo.ListAll(ctx)
	assert.NoError(t, err5)
	assert.Nil(t, result5)
}

// 测试GreeterRepo并发安全性
func TestGreeterRepoConcurrency(t *testing.T) {
	data := &Data{}
	logger := log.NewStdLogger(os.Stdout)

	repo := NewGreeterRepo(data, logger)
	greeterRepo := repo.(*greeterRepo)

	ctx := context.Background()

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			greeter := &biz.Greeter{Hello: "hello" + string(rune(id))}
			result, err := greeterRepo.Save(ctx, greeter)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
