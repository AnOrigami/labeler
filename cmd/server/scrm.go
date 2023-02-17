package server

import "go-admin/app/scrm/router"

func init() {
	AppRouters = append(AppRouters, router.InitRouter)
}
