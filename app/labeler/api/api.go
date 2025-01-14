package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/xuri/excelize/v2"
	"go-admin/app/labeler/service"
	"go-admin/common/actions"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"mime/multipart"
)

type (
	GinHandler      = func(c *gin.Context)
	RouterNoAuth    = func(g *gin.RouterGroup, api *LabelerAPI)
	RouterCheckRole = func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwtauth.GinJWTMiddleware)
)

type LabelerAPI struct {
	LabelerService *service.LabelerService
}

func NewLabelerAPI(svc *service.LabelerService) *LabelerAPI {
	return &LabelerAPI{
		LabelerService: svc,
	}
}

var (
	routerNoAuth    = make([]RouterNoAuth, 0)
	routerCheckRole = make([]RouterCheckRole, 0)
)

func InitRouter(r *gin.Engine, api *LabelerAPI, authMiddleware *jwtauth.GinJWTMiddleware) {
	noAuth := r.Group("")
	for _, f := range routerNoAuth {
		f(noAuth, api)
	}
	auth := r.Group("")
	auth.Use(authMiddleware.MiddlewareFunc(), actions.PermissionAction())
	for _, f := range routerCheckRole {
		f(auth, api, authMiddleware)
	}
}

func QueryObjectID(c *gin.Context) (primitive.ObjectID, error) {
	var req struct {
		ID string `form:"id"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger().WithContext(c.Request.Context()).Error(err.Error())
		return primitive.NilObjectID, err
	}
	oid, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		log.Logger().WithContext(c.Request.Context()).Error(err.Error())
		return primitive.NilObjectID, err
	}
	return oid, nil
}

func ReadFileHeader(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ReadFileHeaderExcel(fh *multipart.FileHeader) ([][]string, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	excelFile, err := excelize.OpenReader(f)
	if err != nil {
		return nil, err
	}
	if excelFile.SheetCount <= 0 {
		return nil, errors.New("读取excel文件表失败")
	}
	return excelFile.GetRows(excelFile.GetSheetName(0))
}
