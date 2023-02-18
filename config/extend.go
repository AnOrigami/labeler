package config

var ExtConfig Extend

// Extend 扩展配置
//
//	extend:
//	  demo:
//	    name: demo-name
//
// 使用方法： config.ExtConfig......即可！！
type Extend struct {
	AMap             AMap                   // 这里配置对应配置文件的结构即可
	CTIRedis         CTIRedisConfig         `yaml:"ctiredis"`
	LocalRedis       RedisConfig            `yaml:"localredis"`
	UptraceDSN       string                 `yaml:"uptracedsn"`
	Modules          Modules                `yaml:"modules"`
	ExportFiles      string                 `yaml:"exportfiles"`
	AudioPrefix      string                 `yaml:"audioprefix"`
	CTIManager       CTIManagerConfig       `yaml:"ctimanager"`
	CacheSentence    CacheSentenceConfig    `yaml:"cachesentence"`
	WeComInteractive WeComInteractiveConfig `yaml:"wecominteractive"`
	MinIO            MinIOConfig            `yaml:"minio"`
	Mongodb          MongodbConfig          `yaml:"mongodb"`
}

type AMap struct {
	Key string
}

type CTIRedisConfig struct {
	RedisConfig          `yaml:",inline"`
	PushOrderKey         string `yaml:"pushorderkey"`
	PullCDRKey           string `yaml:"pullcdrkey"`
	PullCallerChannelKey string `yaml:"pullcallerchannelkey"`
}

type RedisConfig struct {
	Dsn      string `yaml:"dsn"`
	DB       int    `yaml:"db"`
	Password string `yaml:"password"`
}

type UptraceConfig struct {
	DSN string `yaml:"dsn"`
}

type Modules struct {
	CTIManager bool `yaml:"ctimanager"`
}

type CTIManagerConfig struct {
	MaxRobot       int64 `yaml:"maxrobot"`
	MaxCTIQueueLen int64 `yaml:"maxctiqueuelen"`
	Threshold      int64 `yaml:"threshold"`
}

type CacheSentenceConfig struct {
	LocalRedisKey       string `yaml:"localrediskey"`
	MaxSentenceQueueLen int    `yaml:"maxsentencequeuelen"`
	Timeout             int64  `yaml:"timeout"`
}

type WeComInteractiveConfig struct {
	AppKey            string `yaml:"appkey"`
	AppSecret         string `yaml:"appsecret"`
	ValidationMessage string `yaml:"validationmessage"`
}

type MinIOConfig struct {
	Endpoint         string `yaml:"endpoint"`
	Key              string `yaml:"key"`
	Secret           string `yaml:"secret"`
	ExportFileBucket string `yaml:"exportfilebucket"`
}

type MongodbConfig struct {
	DSN       string `yaml:"dsn"`
	LabelerDB string `yaml:"labelerdb"`
}
