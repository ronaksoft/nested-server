package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
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
	"git.ronaksoft.com/nested/server/cmd/server-ntfy/client"
	"git.ronaksoft.com/nested/server/model"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"github.com/kataras/iris/websocket"
	"go.uber.org/zap"
)

var (
	_WelcomeMsgBytes []byte
)

// GatewayServer
type GatewayServer struct {
	wg    *sync.WaitGroup
	ws    *websocket.Server
	iris  *iris.Application
	model *nested.Manager
	ntfy  *ntfy.Client
	file  *file.Server
	api   *api.API
}

func NewGatewayServer() *GatewayServer {
	gateway := new(GatewayServer)

	// Set Welcome Message to send to clients when they connect
	_WelcomeMsg := nested.M{
		"type": "r",
		"data": nested.M{
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
		gateway.model = model
	}

	// Initialize Websocket API
	gateway.ws = websocket.New(websocket.Config{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		MaxMessageSize:    1 * 1024 * 1024,
		EnableCompression: true,
		HandshakeTimeout:  30 * time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      5 * time.Minute,
	})
	gateway.ws.OnConnection(gateway.websocketOnConnection)

	// Initialize NTFY Client
	gateway.ntfy = ntfy.NewClient(_Config.GetString("JOB_ADDRESS"), gateway.model)

	// If a push message received from NATS then send it to the response channel for
	// delivery to end-user
	gateway.ntfy.OnWebsocketPush(func(push *ntfy.WebsocketPush) {
		if gateway.ws.IsConnected(push.WebsocketID) {
			conn := gateway.ws.GetConnection(push.WebsocketID)
			conn.EmitMessage([]byte(push.Payload))
		} else {
			gateway.model.Websocket.Remove(push.WebsocketID, push.BundleID)
		}
	})

	// Remove all the websockets
	gateway.model.Websocket.RemoveByBundleID(_Config.GetString("BUNDLE_ID"))

	// Initialize IRIS Framework
	gateway.iris = iris.New()
	gateway.iris.Use(
		cors.New(cors.Options{
			AllowedHeaders: []string{
				"origin", "access-control-allow-origin", "content-type",
				"accept", "cache-control", "x-file-type", "x-file-size", "x-requested-with",
			},
			AllowOriginFunc: func(origin string) bool {
				return true
			},
		}))

	gateway.wg = new(sync.WaitGroup)

	// Initialize API API
	gateway.api = api.NewServer(_Config, gateway.wg)

	// Register all the available services in the server worker
	gateway.api.Worker().RegisterService(nestedServiceAccount.NewAccountService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceApp.NewAppService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceAdmin.NewAdminService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceAuth.NewAuthService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceHook.NewHookService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceClient.NewClientService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceContact.NewContactService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceFile.NewFileService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceLabel.NewLabelService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceNotification.NewNotificationService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServicePlace.NewPlaceService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServicePost.NewPostService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceReport.NewReportService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceSearch.NewSearchService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceSession.NewSessionService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceSystem.NewSystemService(gateway.api.Worker()))
	gateway.api.Worker().RegisterService(nestedServiceTask.NewTaskService(gateway.api.Worker()))

	// Register and run BackgroundWorkers
	gateway.api.RegisterBackgroundJob(api.NewBackgroundJob(gateway.api, 1*time.Minute, api.JobReporter))
	gateway.api.RegisterBackgroundJob(api.NewBackgroundJob(gateway.api, 1*time.Minute, api.JobOverdueTasks))
	gateway.api.RegisterBackgroundJob(api.NewBackgroundJob(gateway.api, 1*time.Hour, api.JobLicenseManager))

	// Initialize File API
	gateway.file = file.NewServer(_Config, gateway.model)

	// Root Handlers (Deprecated)
	gateway.iris.Get("/", gateway.httpOnConnection)
	gateway.iris.Post("/", gateway.httpOnConnection)

	// API Handlers
	apiParty := gateway.iris.Party("/api")
	apiParty.Get("/check_auth", gateway.httpCheckAuth)
	apiParty.Get("/", gateway.httpOnConnection)
	apiParty.Post("/", gateway.httpOnConnection)

	// File Handlers
	fileParty := gateway.iris.Party("/file")
	fileParty.Get("/view/{fileToken:string}", gateway.file.ServeFileByFileToken, gateway.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}", gateway.file.ServePublicFiles, gateway.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}/{downloadToken:string}", gateway.file.ServePrivateFiles, gateway.file.Download)
	fileParty.Get("/download/{fileToken:string}", gateway.file.ForceDownload, gateway.file.ServeFileByFileToken, gateway.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}", gateway.file.ForceDownload, gateway.file.ServePublicFiles, gateway.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}/{downloadToken:string}", gateway.file.ForceDownload, gateway.file.ServePrivateFiles, gateway.file.Download)
	fileParty.Post("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", gateway.file.UploadUser)
	fileParty.Options("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", nil)
	fileParty.Post("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", gateway.file.UploadApp)
	fileParty.Options("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", nil)

	// System Handlers
	systemParty := gateway.iris.Party("/system")
	systemParty.Get("/download/{apiKey:string}/{universalID:string}", gateway.file.ServerFileBySystem, gateway.file.Download)
	systemParty.Post("/upload/{uploadType:string}/{apiKey:string}", gateway.file.UploadSystem)

	return gateway
}

// Run
// This is a blocking function which will run the Iris server
func (gw *GatewayServer) Run() {
	// Run API
	if _Config.GetString("TLS_KEY_FILE") != "" && _Config.GetString("TLS_CERT_FILE") != "" {
		gw.iris.Run(iris.TLS(
			_Config.GetString("BIND_ADDRESS"),
			_Config.GetString("TLS_CERT_FILE"),
			_Config.GetString("TLS_KEY_FILE"),
		))
	} else {
		gw.iris.Run(iris.Addr(
			_Config.GetString("BIND_ADDRESS"),
		))
	}
}

// Shutdown
func (gw *GatewayServer) Shutdown() {
	gw.model.Shutdown()
	gw.ntfy.Close()
}

// httpOnConnection
// This function is called with any request from clients. If the request has "Upgrade" header set to "websocket"
// then context will be passed to 'websocketOnConnection'
func (gw *GatewayServer) httpOnConnection(ctx iris.Context) {
	startTime := time.Now()
	upgrade := ctx.GetHeader("Upgrade")
	if strings.ToLower(upgrade) == "websocket" {
		ctx.Do([]iris.Handler{gw.ws.Handler()})
		return
	}

	userRequest := new(nestedGateway.Request)
	if err := ctx.ReadJSON(userRequest); err != nil {
		ctx.JSON(iris.Map{
			"status":     "err",
			"error_code": nested.ERR_INVALID,
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

	// Send to API API
	userResponse := new(nestedGateway.Response)
	gw.api.Worker().Execute(userRequest, userResponse)

	_Log.Info("HTTP Request Received",
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
func (gw *GatewayServer) httpCheckAuth(ctx iris.Context) {
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
func (gw *GatewayServer) websocketOnConnection(c websocket.Connection) {
	_Log.Debug("Websocket Connected",
		zap.String("ConnID", c.ID()),
		zap.String("RemoteIP", c.Context().Request().RemoteAddr),
	)

	// Send Welcome Message to the Client
	_ = c.EmitMessage(_WelcomeMsgBytes)

	// Websocket Message Handler
	c.OnMessage(func(m []byte) {
		if strings.HasPrefix(string(m), "PING!") {
			_ = c.EmitMessage([]byte(strings.Replace(string(m), "PING!", "PONG!", 1)))
		} else {
			startTime := time.Now()
			userRequest := new(nestedGateway.Request)
			_ = json.Unmarshal(m, userRequest)
			userRequest.ClientIP = c.Context().RemoteAddr()
			userRequest.UserAgent = c.Context().GetHeader("User-Agent")
			userRequest.WebsocketID = c.ID()

			// Send to API API
			userResponse := new(nestedGateway.Response)
			gw.api.Worker().Execute(userRequest, userResponse)
			_Log.Debug("Websocket Request Received",
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

	// Websocket Disconnect Handler
	c.OnDisconnect(func() {
		gw.model.Websocket.Remove(c.ID(), _BundleID)
	})
}
