package main

import (
	"encoding/json"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/account"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/admin"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/app"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/auth"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/contact"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/file"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/hook"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/label"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/notification"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/place"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/post"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/report"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/search"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/session"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/system"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api/task"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_file"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	_WelcomeMsgBytes []byte
)

type APP struct {
	wg    *sync.WaitGroup
	ws    *websocket.Server
	iris  *iris.Application
	model *nested.Manager
	file  *file.Server
	api   *api.Server
}

func NewAPP() *APP {
	app := new(APP)

	// Set Welcome Message to send to clients when they connect
	_WelcomeMsg := tools.M{
		"type": "r",
		"data": tools.M{
			"status": "ok",
			"msg":    "hi",
		},
	}
	_WelcomeMsgBytes, _ = json.Marshal(_WelcomeMsg)

	// Initialize Nested Model
	if model, err := nested.NewManager(
		_Config.GetString("INSTANCE_ID"),
		_Config.GetString("MONGO_DSN"),
		_Config.GetString("REDIS_DSN"),
		_Config.GetInt("DEBUG_LEVEL"),
	); err != nil {
		os.Exit(1)
	} else {
		app.model = model
	}

	// Initialize websocket Server
	app.ws = websocket.New(websocket.Config{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		MaxMessageSize:    1 * 1024 * 1024,
		EnableCompression: true,
		HandshakeTimeout:  30 * time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      5 * time.Minute,
	})
	app.ws.OnConnection(app.websocketOnConnection)

	// Initialize IRIS Framework
	app.iris = iris.New()
	app.iris.Use(
		cors.New(cors.Options{
			AllowedHeaders: []string{
				"origin", "access-control-allow-origin", "content-type",
				"accept", "cache-control", "x-file-type", "x-file-size", "x-requested-with",
			},
			AllowOriginFunc: func(origin string) bool {
				return true
			},
		}))

	app.wg = new(sync.WaitGroup)

	// Initialize Server Server
	app.api = api.NewServer(_Config, app.wg,
		func(push pusher.WebsocketPush) bool {
			if app.ws.IsConnected(push.WebsocketID) {
				conn := app.ws.GetConnection(push.WebsocketID)
				_ = conn.EmitMessage([]byte(push.Payload))
				return true
			}
			return false
		},
	)

	// Register all the available services in the server worker
	app.api.Worker().RegisterService(
		nestedServiceAccount.NewAccountService(app.api.Worker()),
		nestedServiceApp.NewAppService(app.api.Worker()),
		nestedServiceAdmin.NewAdminService(app.api.Worker()),
		nestedServiceAuth.NewAuthService(app.api.Worker()),
		nestedServiceHook.NewHookService(app.api.Worker()),
		nestedServiceClient.NewClientService(app.api.Worker()),
		nestedServiceContact.NewContactService(app.api.Worker()),
		nestedServiceFile.NewFileService(app.api.Worker()),
		nestedServiceLabel.NewLabelService(app.api.Worker()),
		nestedServiceNotification.NewNotificationService(app.api.Worker()),
		nestedServicePlace.NewPlaceService(app.api.Worker()),
		nestedServicePost.NewPostService(app.api.Worker()),
		nestedServiceReport.NewReportService(app.api.Worker()),
		nestedServiceSearch.NewSearchService(app.api.Worker()),
		nestedServiceSession.NewSessionService(app.api.Worker()),
		nestedServiceSystem.NewSystemService(app.api.Worker()),
		nestedServiceTask.NewTaskService(app.api.Worker()),
	)

	// Register and run BackgroundWorkers
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Minute, api.JobReporter))
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Minute, api.JobOverdueTasks))
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Hour, api.JobLicenseManager))

	// Initialize File Server
	app.file = file.NewServer(_Config, app.model)

	// Root Handlers (Deprecated)
	app.iris.Get("/", app.httpOnConnection)
	app.iris.Post("/", app.httpOnConnection)

	// Server Handlers
	apiParty := app.iris.Party("/api")
	apiParty.Get("/check_auth", app.httpCheckAuth)
	apiParty.Get("/", app.httpOnConnection)
	apiParty.Post("/", app.httpOnConnection)

	// File Handlers
	fileParty := app.iris.Party("/file")
	fileParty.Get("/view/{fileToken:string}", app.file.ServeFileByFileToken, app.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}", app.file.ServePublicFiles, app.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}/{downloadToken:string}", app.file.ServePrivateFiles, app.file.Download)
	fileParty.Get("/download/{fileToken:string}", app.file.ForceDownload, app.file.ServeFileByFileToken, app.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}", app.file.ForceDownload, app.file.ServePublicFiles, app.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}/{downloadToken:string}", app.file.ForceDownload, app.file.ServePrivateFiles, app.file.Download)
	fileParty.Post("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", app.file.UploadUser)
	fileParty.Options("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", nil)
	fileParty.Post("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", app.file.UploadApp)
	fileParty.Options("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", nil)

	// System Handlers
	systemParty := app.iris.Party("/system")
	systemParty.Get("/download/{apiKey:string}/{universalID:string}", app.file.ServerFileBySystem, app.file.Download)
	systemParty.Post("/upload/{uploadType:string}/{apiKey:string}", app.file.UploadSystem)

	return app
}

// Run
// This is a blocking function which will run the Iris server
func (gw *APP) Run() {
	// Run Server
	if _Config.GetString("TLS_KEY_FILE") != "" && _Config.GetString("TLS_CERT_FILE") != "" {
		_ = gw.iris.Run(iris.TLS(
			_Config.GetString("BIND_ADDRESS"),
			_Config.GetString("TLS_CERT_FILE"),
			_Config.GetString("TLS_KEY_FILE"),
		))
	} else {
		_ = gw.iris.Run(iris.Addr(
			_Config.GetString("BIND_ADDRESS"),
		))
	}
}

// Shutdown clean up services before exiting
func (gw *APP) Shutdown() {
	gw.model.Shutdown()
}

// httpOnConnection
// This function is called with any request from clients. If the request has "Upgrade" header set to "websocket"
// then context will be passed to 'websocketOnConnection'
func (gw *APP) httpOnConnection(ctx iris.Context) {
	startTime := time.Now()
	upgrade := ctx.GetHeader("Upgrade")
	if strings.ToLower(upgrade) == "websocket" {
		ctx.Do([]iris.Handler{gw.ws.Handler()})
		return
	}

	userRequest := new(rpc.Request)
	if err := ctx.ReadJSON(userRequest); err != nil {
		ctx.JSON(iris.Map{
			"status":     "err",
			"error_code": global.ErrInvalid,
			"err_items":  []string{"not_valid_json"},
		})
		return
	}
	userRequest.ClientIP = ctx.RemoteAddr()
	userRequest.UserAgent = ctx.GetHeader("User-Agent")
	if appID := ctx.GetHeader("X-APP-ID"); len(appID) > 0 {
		userRequest.AppID = appID
	}
	if appToken := ctx.GetHeader("X-APP-TOKEN"); len(appToken) > 0 {
		userRequest.AppToken = appToken
	}

	// Send to Server
	userResponse := new(rpc.Response)
	gw.api.Worker().Execute(userRequest, userResponse)

	log.Info("HTTP Request Received",
		zap.String("AppID", userRequest.AppID),
		zap.String("Cmd", userRequest.Command),
		zap.String("Status", userResponse.Status),
		zap.Duration("Duration", time.Now().Sub(startTime)),
		zap.Any("Response", userResponse.Data),
	)

	responseBytes, _ := json.Marshal(userResponse)
	n, _ := ctx.Write(responseBytes)
	gw.model.Report.CountDataOut(n)
}

// httpCheckAuth
func (gw *APP) httpCheckAuth(ctx iris.Context) {
	appToken := gw.model.Token.GetAppToken(ctx.GetHeader("X-APP-TOKEN"))
	if appToken != nil && !appToken.Expired {
		app := gw.model.App.GetByID(ctx.GetHeader("X-APP-ID"))
		if app != nil && appToken.AppID == app.ID {
			account := gw.model.Account.GetByID(appToken.AccountID, nil)
			ctx.StatusCode(http.StatusOK)
			ctx.JSON(iris.Map{
				"account_id": account.ID,
				"name":       account.FullName,
			})
			return
		}
	}
	ctx.StatusCode(http.StatusForbidden)
	return
}

// websocketOnConnection
// This function will be called once in each websocket connection life-time
func (gw *APP) websocketOnConnection(c websocket.Connection) {
	log.Debug("websocket Connected",
		zap.String("ConnID", c.ID()),
		zap.String("RemoteIP", c.Context().Request().RemoteAddr),
	)

	// Send Welcome Message to the Client
	_ = c.EmitMessage(_WelcomeMsgBytes)

	// websocket Message Handler
	c.OnMessage(func(m []byte) {
		if strings.HasPrefix(string(m), "PING!") {
			_ = c.EmitMessage([]byte(strings.Replace(string(m), "PING!", "PONG!", 1)))
		} else {
			startTime := time.Now()
			userRequest := &rpc.Request{}
			_ = json.Unmarshal(m, userRequest)
			userRequest.ClientIP = c.Context().RemoteAddr()
			userRequest.UserAgent = c.Context().GetHeader("User-Agent")
			userRequest.WebsocketID = c.ID()

			// Send to Server
			userResponse := &rpc.Response{}
			gw.api.Worker().Execute(userRequest, userResponse)
			log.Debug("websocket Request Received",
				zap.String("AppID", userRequest.AppID),
				zap.String("Cmd", userRequest.Command),
				zap.String("Status", userResponse.Status),
				zap.Duration("Duration", time.Now().Sub(startTime)),
			)
			bytes, _ := json.Marshal(userResponse)
			_ = c.EmitMessage(bytes)
			gw.model.Report.CountDataOut(len(bytes))
		}
	})

	// websocket Disconnect Handler
	c.OnDisconnect(func() {
		_ = gw.api.Worker().Pusher().UnregisterWebsocket(c.ID(), _BundleID)
	})
}