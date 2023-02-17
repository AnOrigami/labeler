package admin

import (
	"context"
	"fmt"
	"github.com/go-admin-team/go-admin-core/config/source/file"
	"go-admin/app/admin/models"
	"go-admin/app/scrm"
	"go-admin/common/database"
	"go-admin/common/log"
	"go-admin/common/storage"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/spf13/cobra"

	"go-admin/app/admin/router"
	"go-admin/app/jobs"
	"go-admin/common/global"
	common "go-admin/common/middleware"
	ext "go-admin/config"
)

const ServiceName = "scrm_admin"

var (
	configYml string
	StartCmd  = &cobra.Command{
		Use:          "admin",
		Short:        "Start admin server",
		Example:      "go-admin admin -c config/settings.yml",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

var AppRouters = make([]func(), 0)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/settings.yml", "Start server with provided configuration file")

	//注册路由 fixme 其他应用的路由，在本目录新建文件放在init方法
	AppRouters = append(AppRouters, router.InitRouter)
}

func run() error {
	_ = log.WithTracer(startingCtx, PackageName, "注入配置扩展项", func(ctx context.Context) error {
		config.ExtendConfig = &ext.ExtConfig
		//1. 读取配置
		config.Setup(
			file.NewSource(file.WithPath(configYml)),
			database.Setup,
			storage.Setup,
		)
		return nil
	})

	_ = log.WithTracer(startingCtx, PackageName, "setup 注册监听函数", func(ctx context.Context) error {
		queue := sdk.Runtime.GetMemoryQueue("")
		queue.Register(global.LoginLog, models.SaveLoginLog)
		queue.Register(global.OperateLog, models.SaveOperaLog)
		go queue.Run()
		return nil
	})

	if config.ApplicationConfig.Mode == pkg.ModeProd.String() {
		gin.SetMode(gin.ReleaseMode)
	}
	initRouter()

	for _, f := range AppRouters {
		f()
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ApplicationConfig.Host, config.ApplicationConfig.Port),
		Handler: sdk.Runtime.GetEngine(),
	}

	go func() {
		jobs.InitJob()
		jobs.Setup(sdk.Runtime.GetDb())
	}()

	go func() {
		// 服务连接
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Logger().Fatal("listen: ", err)
		}
	}()
	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Printf("%s Shutdown Server ... \r\n", pkg.GetCurrentTimeStr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Logger().Fatal("Server Shutdown:", err)
	}
	log.Logger().Println("Server exiting")

	return nil
}

func initRouter() {
	var r *gin.Engine
	h := sdk.Runtime.GetEngine()
	if h == nil {
		h = gin.New()
		sdk.Runtime.SetEngine(h)
	}
	if v, ok := h.(*gin.Engine); ok {
		r = v
	} else {
		log.Logger().Fatal("not support other engine")
	}
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), scrm.GinContextKey, c))
	})
	r.Use(otelgin.Middleware(ServiceName))
	r.
		Use(common.Sentinel()).
		Use(common.RequestId(pkg.TrafficKey)).
		Use(api.SetRequestLogger)

	common.InitMiddleware(r)
}
