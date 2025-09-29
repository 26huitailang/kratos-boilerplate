package benchmark

import (
	"context"
	"io"
	"testing"

	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/pkg/errors"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2/log"
)

// SimpleGreeterRepo 简单的模拟仓库，用于基准测试
type SimpleGreeterRepo struct{}

func (r *SimpleGreeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	return &biz.Greeter{Hello: g.Hello}, nil
}

func (r *SimpleGreeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	return g, nil
}

func (r *SimpleGreeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	return &biz.Greeter{Hello: "found"}, nil
}

func (r *SimpleGreeterRepo) ListByHello(ctx context.Context, hello string) ([]*biz.Greeter, error) {
	return []*biz.Greeter{{Hello: hello}}, nil
}

func (r *SimpleGreeterRepo) ListAll(ctx context.Context) ([]*biz.Greeter, error) {
	return []*biz.Greeter{{Hello: "all"}}, nil
}

// BenchmarkGreeterService Greeter服务性能基准测试
func BenchmarkGreeterService(b *testing.B) {
	logger := log.NewStdLogger(io.Discard)
	repo := &SimpleGreeterRepo{}
	uc := biz.NewGreeterUsecase(repo, logger)
	srv := service.NewGreeterService(uc)

	ctx := context.Background()

	b.Run("SayHello_Normal", func(b *testing.B) {
		req := &v1.HelloRequest{Name: "world"}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			srv.SayHello(ctx, req)
		}
	})

	b.Run("SayHello_ErrorScenarios", func(b *testing.B) {
		scenarios := []string{"error", "badrequest", "notfound", "forbidden", "timeout"}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			scenario := scenarios[i%len(scenarios)]
			req := &v1.HelloRequest{Name: scenario}
			srv.SayHello(ctx, req)
		}
	})

	b.Run("SayHello_Concurrent", func(b *testing.B) {
		req := &v1.HelloRequest{Name: "world"}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				srv.SayHello(ctx, req)
			}
		})
	})
}

// BenchmarkErrorHandling 错误处理性能基准测试
func BenchmarkErrorHandling(b *testing.B) {
	b.Run("ErrorCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errors.NewError(errors.ErrCodeInvalidParams, "test error")
		}
	})

	b.Run("ErrorWrapping", func(b *testing.B) {
		baseErr := errors.InternalError("base error")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			errors.WrapError(baseErr, errors.ErrCodeInternalError, "wrapped error")
		}
	})

	b.Run("ErrorCollector", func(b *testing.B) {
		err1 := errors.InvalidParams("param1")
		err2 := errors.InvalidParams("param2")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			collector := errors.NewErrorCollector()
			collector.Add(err1)
			collector.Add(err2)
			collector.ToMultiError()
		}
	})
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	logger := log.NewStdLogger(io.Discard)
	repo := &SimpleGreeterRepo{}
	uc := biz.NewGreeterUsecase(repo, logger)
	srv := service.NewGreeterService(uc)
	ctx := context.Background()

	b.Run("SmallPayload", func(b *testing.B) {
		req := &v1.HelloRequest{Name: "world"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			srv.SayHello(ctx, req)
		}
	})

	b.Run("LargePayload", func(b *testing.B) {
		// 创建大载荷（1KB字符串）
		largePayload := string(make([]byte, 1024))
		req := &v1.HelloRequest{Name: largePayload}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			srv.SayHello(ctx, req)
		}
	})
}
