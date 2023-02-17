package server

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"go-admin/common/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/config/source/file"
	"github.com/go-admin-team/go-admin-core/sdk"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/spf13/cobra"

	"go-admin/common/database"
	common "go-admin/common/middleware"
	"go-admin/common/middleware/handler"
	"go-admin/common/storage"
	ext "go-admin/config"
)

const ServiceName = "scrm_server"

var (
	configYml string
	StartCmd  = &cobra.Command{
		Use:          "server",
		Short:        "Start API server",
		Example:      "go-admin server -c config/settings.yml",
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			setup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

var AppRouters = make([]func(), 0)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/settings.yml", "Start server with provided configuration file")

	//注册路由 fixme 其他应用的路由，在本目录新建文件放在init方法
	//AppRouters = append(AppRouters, router.InitRouter)
}

func setup() {
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

	_ = log.WithTracer(startingCtx, PackageName, "初始化MinIO", func(ctx context.Context) error {
		cfg := ext.ExtConfig.MinIO
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds: credentials.NewStaticV4(cfg.Key, cfg.Secret, ""),
		})
		if err != nil {
			scrm.Logger().WithContext(ctx).Fatal(err)
		}
		scrm.MinIOClient = client
		return nil
	})

	var ctiRedisClient *redis.Client
	_ = log.WithTracer(startingCtx, PackageName, "setup 初始化CTI Redis", func(ctx context.Context) error {
		scrm.Logger().WithContext(ctx).Info("初始化CTI Redis")
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		{
			ctiRedisClient = redis.NewClient(&redis.Options{
				Addr:     ext.ExtConfig.CTIRedis.Dsn,
				Password: ext.ExtConfig.CTIRedis.Password,
				DB:       ext.ExtConfig.CTIRedis.DB,
			})
			ctiRedisClient.AddHook(redisotel.NewTracingHook())
			_, err := ctiRedisClient.Ping(ctx).Result()
			if err != nil {
				scrm.Logger().WithContext(ctx).Fatal(err)
			}
			scrm.CtiRedisClient = ctiRedisClient
		}
		return nil
	})

	var localRedisClient *redis.Client
	_ = log.WithTracer(startingCtx, PackageName, "setup 初始化本地Redis", func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		{
			localRedisClient = redis.NewClient(&redis.Options{
				Addr:     ext.ExtConfig.LocalRedis.Dsn,
				Password: ext.ExtConfig.LocalRedis.Password,
				DB:       ext.ExtConfig.LocalRedis.DB,
			})
			localRedisClient.AddHook(redisotel.NewTracingHook())
			_, err := localRedisClient.Ping(ctx).Result()
			if err != nil {
				scrm.Logger().WithContext(ctx).Fatal(err)
			}
			scrm.RedisClient = localRedisClient
		}
		return nil
	})

	_ = log.WithTracer(startingCtx, PackageName, "setup 初始化GORM", func(ctx context.Context) error {
		scrm.Logger().WithContext(ctx).Info("初始化GORM")
		scrm.GormDB = sdk.Runtime.GetDbByKey("")
		if ext.ExtConfig.UptraceDSN != "" {
			if err := scrm.GormDB.Use(otelgorm.NewPlugin()); err != nil {
				scrm.Logger().WithContext(ctx).Fatal(err)
			}
		}
		return nil
	})

	_ = log.WithTracer(startingCtx, PackageName, "setup CTIManager", func(ctx context.Context) error {
		if ext.ExtConfig.Modules.CTIManager {
			scrm.Logger().WithContext(ctx).Info("CTIManager starting")
			if len(ext.ExtConfig.CTIRedis.PushOrderKey) == 0 ||
				len(ext.ExtConfig.CTIRedis.PullCDRKey) == 0 ||
				len(ext.ExtConfig.CTIRedis.PullCallerChannelKey) == 0 {
				scrm.Logger().WithContext(ctx).Fatal("cti config error")
			}
			ctiManager := &service.CTIManager{
				Ctx:                  context.Background(),
				MaxRobotCon:          ext.ExtConfig.CTIManager.MaxRobot,
				MaxCTIQueueLen:       ext.ExtConfig.CTIManager.MaxCTIQueueLen,
				Threshold:            ext.ExtConfig.CTIManager.Threshold,
				PushOrderKey:         ext.ExtConfig.CTIRedis.PushOrderKey,
				PullCDRKey:           ext.ExtConfig.CTIRedis.PullCDRKey,
				PullCallerChannelKey: ext.ExtConfig.CTIRedis.PullCallerChannelKey,
				GormDB:               sdk.Runtime.GetDbByKey(""),
				CTIRDB:               ctiRedisClient,
				LocalRDB:             localRedisClient,
			}
			ctiManager.Run()
			scrm.Logger().WithContext(ctx).Info("CTIManager started")
		}
		return nil
	})

	_ = log.WithTracer(startingCtx, PackageName, "setup seat service statistic", func(ctx context.Context) error {
		scrm.Logger().WithContext(ctx).Info("seat statistic service starting")
		service.SeatStatSvc = &service.MemSeatStatisticService{
			SeatInfoMap: &map[int]service.SeatStatisticInfo{},
			Lock:        sync.Mutex{},
			LoopTime:    1 * time.Minute,
		}
		go service.SeatStatSvc.Run()
		scrm.Logger().WithContext(ctx).Info("seat statistic service started")
		return nil
	})

	_ = log.WithTracer(startingCtx, PackageName, "init QuanLiang token", func(ctx context.Context) error {
		service.QuanLiangSessionInit(ctx)
		return nil
	})
}

func run() error {
	_ = log.WithTracer(startingCtx, PackageName, "starting run", func(ctx context.Context) error {
		scrm.Logger().WithContext(ctx).Info("starting run")
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
	if ext.ExtConfig.UptraceDSN != "" {
		r.Use(otelgin.Middleware(ServiceName))
	}
	if config.SslConfig.Enable {
		r.Use(handler.TlsHandler())
	}
	//r.Use(middleware.Metrics())
	r.
		Use(common.Sentinel()).
		Use(common.RequestId(pkg.TrafficKey)).
		Use(api.SetRequestLogger)

	common.InitMiddleware(r)
}
