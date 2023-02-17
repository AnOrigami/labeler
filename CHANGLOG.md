## 1.11.1

异步导出的通话记录保存位置修改为minio; `settings.yml`新增`extend.minio`(`server`和`export_task`需要);

**更新部署**

更新`server`和`export_task`, 注意要给`server`和`export_task`增加配置:

```
      endpoint: '...'
      key: '...'
      secret: '...'
      exportfilebucket: '...'
```

## 1.11.0-r1

把`go-admin`提供的api从`server`命令拆分到新增的`admin`命令中; 路由规则调整成`/api/v1/scrm/ --> server`, `/api/ --> admin`;

**更新部署**

1. 使用`admin -c settings.yml`启动`admin`服务;
2. 更新`server`服务;
3. 修改反向代理的路由规则;

## 1.11.0

将异步导出功能（export_task）分离为单独的服务, 新增一个服务入口;需要停止并更新scrm server服务然后重启;启动异步导出服务的参数为 export_task -c config/settings.yml;

## 1.10.4

关闭上传工单时的手机号码校验;

## 1.10.3

坐席签入/迁出/示忙/示闲时把preready设置为false;

## 1.10.2

CTI PushOrder计算并发时排除没有签入的坐席;

## 1.10.1

设置坐席预示闲接口同步发送ws状态

## 1.10.0

增加设置坐席preready配置接口

## 1.9.5

修复导出时间精度字段赋值问题

## 1.9.4

修复企微绑定的数据库更新语句;

## 1.9.3

企业微信交互增加解绑功能

## 1.9.2

bugfix: 大批量导出的时候缺失数据的问题

## 1.9.1

**migrate** order表的phone类型改为longtext; seat表的两个微信相关字段改为varchar(127);

## 1.9.0

feature: 增加企业微信的交互功能
 - 执行 `migrate` (1669082543506)
 - 配置文件增加 `wecominteractive` 配置,包括两个密钥和加好友的问候语

## v1.8.12

bugfix: 修复sync_sentence: 解决flag导致的循环次数错误;

## v1.8.11

bugfix: 坐席示闲时检查是否签入+是否有选择项目;

## v1.8.10

锁定坐席时给坐席示忙

## V1.8.9

导出时间统一改成毫秒级精度

## v1.8.8

为 `export_task` 增加了通用的分批查询功能

## v.1.8.7

搜索通话记录和导出通话记录接口增加seatID筛选条件

## v1.8.6

修改项目获取坐席详情锁定次数和通话时间时数据不一致的问题

## v1.8.5

修复无法取消项目中所有坐席的问题和新增获取当前坐席相关的项目列表

## v1.8.4

修复导出任务返回信息为用户友好型