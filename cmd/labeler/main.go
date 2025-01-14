package labeler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/config/source/file"
	"github.com/go-admin-team/go-admin-core/sdk"
	sdkapi "github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/spf13/cobra"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonoptions"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"gorm.io/gorm"

	"go-admin/app/labeler/api"
	service2 "go-admin/app/labeler/service"
	"go-admin/app/scrm"
	"go-admin/common/database"
	"go-admin/common/log"
	common "go-admin/common/middleware"
	"go-admin/common/storage"
	"go-admin/common/util"
	ext "go-admin/config"
)

const ServiceName = "labeler"

var (
	configYml string
	StartCmd  = &cobra.Command{
		Use:          "labeler",
		Short:        "Start API server",
		Example:      "go-admin labeler -c config/settings.yml",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/settings.yml", "Start server with provided configuration file")

	//注册路由 fixme 其他应用的路由，在本目录新建文件放在init方法
	//AppRouters = append(AppRouters, router.InitRouter)
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

	var gormDB *gorm.DB
	_ = log.WithTracer(startingCtx, PackageName, "初始化GORM", func(ctx context.Context) error {
		gormDB = sdk.Runtime.GetDbByKey("")
		scrm.GormDB = gormDB
		if err := scrm.GormDB.Use(otelgorm.NewPlugin()); err != nil {
			scrm.Logger().WithContext(ctx).Fatal(err)
		}
		return nil
	})

	var mongodbClient *mongo.Client
	_ = log.WithTracer(startingCtx, PackageName, "初始化MongoDB", func(ctx context.Context) error {
		cfg := ext.ExtConfig.Mongodb
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rb := bsoncodec.NewRegistryBuilder()
		bsoncodec.DefaultValueEncoders{}.RegisterDefaultEncoders(rb)
		bsoncodec.DefaultValueDecoders{}.RegisterDefaultDecoders(rb)
		timeCodec := util.NewTimeCodec(bsonoptions.TimeCodec().SetUseLocalTimeZone(true))
		rb.RegisterCodec(reflect.TypeOf(util.Datetime{}), timeCodec)
		rb.RegisterCodec(util.GzipJSONType, &util.JSONCodec{})
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DSN).SetMonitor(otelmongo.NewMonitor()).SetRegistry(rb.Build()))
		if err != nil {
			panic(err)
		}
		mongodbClient = client
		return nil
	})

	service := service2.NewLabelerService(mongodbClient, gormDB)
	labelerAPI := api.NewLabelerAPI(service)

	r := gin.New()
	_ = log.WithTracer(startingCtx, PackageName, "初始化router", func(ctx context.Context) error {
		authMiddleware, err := common.AuthInit()
		if err != nil {
			log.Logger().WithContext(ctx).Fatalf("JWT Init Error, %s", err.Error())
		}
		r.Use(func(c *gin.Context) {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), scrm.GinContextKey, c))
		})
		r.
			Use(common.Sentinel()).
			Use(common.RequestId(pkg.TrafficKey)).
			Use(sdkapi.SetRequestLogger)
		common.InitMiddleware(r)
		r.Use(otelgin.Middleware(ServiceName))
		api.InitRouter(r, labelerAPI, authMiddleware)
		return nil
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.ApplicationConfig.Host, config.ApplicationConfig.Port),
		Handler: r,
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
