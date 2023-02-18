package labeler

import (
	"context"
	"github.com/go-admin-team/go-admin-core/config/source/file"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/spf13/cobra"
	service2 "go-admin/app/labeler/service"
	"go-admin/common/database"
	"go-admin/common/log"
	"go-admin/common/storage"
	ext "go-admin/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

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

	var mongodbClient *mongo.Client
	_ = log.WithTracer(startingCtx, PackageName, "初始化MongoDB", func(ctx context.Context) error {
		cfg := ext.ExtConfig.Mongodb
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DSN))
		if err != nil {
			panic(err)
		}
		mongodbClient = client
		return nil
	})

	service := service2.NewLabelerService(mongodbClient)
	_ = service
	return nil
}
