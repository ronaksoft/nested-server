package main

import (
	"encoding/json"
	"fmt"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/mail/lmtp"
	mailmap "git.ronaksoft.com/nested/server/pkg/mail/map"
	"git.ronaksoft.com/nested/server/pkg/pusher"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/account"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/admin"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/app"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/auth"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/client"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/contact"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/file"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/hook"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/label"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/notification"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/place"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/post"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/report"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/search"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/session"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/system"
	"git.ronaksoft.com/nested/server/pkg/rpc/api/task"
	"git.ronaksoft.com/nested/server/pkg/rpc/file"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/websocket"
	"github.com/kataras/neffos"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	_WelcomeMsgBytes []byte
)

type APP struct {
	systemKey string
	wg        *sync.WaitGroup
	ws        *neffos.Server
	iris      *iris.Application
	model     *nested.Manager
	file      *file.Server
	api       *api.Worker
	mailStore *lmtp.Server
	mailMap   *mailmap.Server
	pusher    *pusher.Pusher
}

func NewAPP() *APP {
	app := &APP{
		systemKey: config.GetString(config.SystemAPIKey),
	}

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
		config.GetString(config.InstanceID),
		config.GetString(config.MongoDSN),
		config.GetString(config.RedisDSN),
		config.GetInt(config.LogLevel),
	); err != nil {
		os.Exit(1)
	} else {
		app.model = model
	}

	// Initialize websocket Server
	app.ws = websocket.New(websocket.DefaultGobwasUpgrader, websocket.Events{
		websocket.OnNativeMessage: app.websocketOnMessage,
	})
	app.ws.OnConnect = app.websocketOnConnect
	app.ws.OnDisconnect = app.websocketOnDisconnect

	// Initialize Pusher
	app.pusher = pusher.New(
		app.model,
		config.GetString(config.BundleID), config.GetString(config.SenderDomain),
	)
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

	// Run Model Checkups
	nested.StartupCheckups()

	// Register Bundle
	app.model.RegisterBundle(config.GetString(config.BundleID))

	app.api = api.NewWorker(app.model, app.pusher)

	// Register all the available services in the server worker
	app.api.RegisterService(
		nestedServiceAccount.NewAccountService,
		nestedServiceApp.NewAppService,
		nestedServiceAdmin.NewAdminService,
		nestedServiceAuth.NewAuthService,
		nestedServiceHook.NewHookService,
		nestedServiceClient.NewClientService,
		nestedServiceContact.NewContactService,
		nestedServiceFile.NewFileService,
		nestedServiceLabel.NewLabelService,
		nestedServiceNotification.NewNotificationService,
		nestedServicePlace.NewPlaceService,
		nestedServicePost.NewPostService,
		nestedServiceReport.NewReportService,
		nestedServiceSearch.NewSearchService,
		nestedServiceSession.NewSessionService,
		nestedServiceSystem.NewSystemService,
		nestedServiceTask.NewTaskService,
	)

	// Register and run BackgroundWorkers
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Minute, api.JobReporter))
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Minute, api.JobOverdueTasks))
	app.api.RegisterBackgroundJob(api.NewBackgroundJob(app.api, 1*time.Hour, api.JobLicenseManager))

	// Initialize File Server
	app.file = file.NewServer(app.model)

	// Initialize Mail Store (LMTP)
	app.mailStore = lmtp.New(app.model, filepath.Join(config.GetString(config.PostfixCHRoot), config.GetString(config.MailStoreSock)))

	// Initialize Mail Map (TCP)
	app.mailMap = mailmap.New(app.model)

	// Server Handlers
	app.iris.Get("/ws", websocket.Handler(app.ws))
	apiParty := app.iris.Party("/api")
	apiParty.Get("/check_auth", app.httpCheckAuth)

	// File Handlers
	fileParty := app.iris.Party("/file")
	fileParty.Get("/view/{fileToken:string}", app.file.ServeFileByFileToken, app.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}", app.file.ServePublicFiles, app.file.Download)
	fileParty.Get("/view/{sessionID:string}/{universalID:string}/{downloadToken:string}", app.file.ServePrivateFiles, app.file.Download)
	fileParty.Get("/download/{fileToken:string}", app.file.ForceDownload, app.file.ServeFileByFileToken, app.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}", app.file.ForceDownload, app.file.ServePublicFiles, app.file.Download)
	fileParty.Get("/download/{sessionID:string}/{universalID:string}/{downloadToken:string}", app.file.ForceDownload, app.file.ServePrivateFiles, app.file.Download)
	fileParty.Post("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", app.file.UploadUser)
	//fileParty.Options("/upload/{uploadType:string}/{sessionID:string}/{uploadToken:string}", nil)
	fileParty.Post("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", app.file.UploadApp)
	//fileParty.Options("/upload/app/{uploadType:string}/{appToken:string}/{uploadToken:string}", nil)

	// System Handlers
	systemParty := app.iris.Party("/system")
	systemParty.Get("/download/{apiKey:string}/{universalID:string}", app.checkSystemKey, app.file.Download)
	systemParty.Post("/upload/{uploadType:string}/{apiKey:string}", app.checkSystemKey, app.file.UploadSystem)
	systemParty.Get("/pusher/place_activity/{apiKey:string}/{placeID:string}/{placeActivity:int}", app.checkSystemKey, app.PushPlaceActivity)
	return app
}

// Run
// This is a blocking function which will run the Iris server
func (gw *APP) Run() {
	log.Info("MailStore Server started",
		zap.String("UploadBaseURL", config.GetString(config.MailUploadBaseURL)),
		zap.String("Unix", gw.mailStore.Addr()),
	)
	gw.mailStore.Run()

	log.Info("MailMap Server started", zap.String("TCP", gw.mailMap.Addr()))
	go func() {
		gw.mailMap.Run()
	}()

	// Run Server
	addr := fmt.Sprintf("%s:%d", config.GetString(config.BindIP), config.GetInt(config.BindPort))

	if config.GetString(config.TlsKeyFile) != "" && config.GetString(config.TlsCertFile) != "" {
		_ = gw.iris.Run(iris.TLS(
			addr,
			config.GetString(config.TlsCertFile),
			config.GetString(config.TlsKeyFile),
		))
	} else {
		_ = gw.iris.Run(iris.Addr(
			addr,
		))
	}

}

// Shutdown clean up services before exiting
func (gw *APP) Shutdown() {
	gw.mailStore.Close()
	gw.model.Shutdown()
}

// httpOnConnection
// This function is called with any request from clients. If the request has "Upgrade" header set to "websocket"
// then context will be passed to 'websocketOnConnection'
func (gw *APP) httpOnConnection(ctx iris.Context) {
	startTime := time.Now()

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
	gw.api.Execute(userRequest, userResponse)

	log.Debug("HTTP Request Received",
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

func (gw *APP) websocketOnConnect(c *websocket.Conn) error {
	log.Debug("Websocket Connected",
		zap.String("ConnID", c.ID()),
		zap.String("RemoteIP", c.Socket().Request().RemoteAddr),
	)

	c.Write(websocket.Message{
		IsNative: true,
		Body:     _WelcomeMsgBytes,
	})

	return nil
}

func (gw *APP) websocketOnDisconnect(c *websocket.Conn) {
	_ = gw.api.Pusher().UnregisterWebsocket(c.ID(), config.GetString(config.BundleID))
}

func (gw *APP) websocketOnMessage(conn *neffos.NSConn, message neffos.Message) error {
	if strings.HasPrefix(string(message.Body), "PING!") {
		conn.Conn.Write(websocket.Message{
			IsNative: true,
			Body:     []byte(strings.Replace(string(message.Body), "PING!", "PONG!", 1)),
		})
	} else {
		startTime := time.Now()
		userRequest := &rpc.Request{}
		_ = json.Unmarshal(message.Body, userRequest)
		userRequest.ClientIP = conn.Conn.Socket().Request().RemoteAddr
		userRequest.UserAgent = conn.Conn.Socket().Request().Header.Get("User-Agent")
		userRequest.WebsocketID = conn.Conn.ID()

		// Send to Server
		userResponse := &rpc.Response{}
		gw.api.Execute(userRequest, userResponse)
		log.Debug("Websocket Request Received",
			zap.String("AppID", userRequest.AppID),
			zap.String("Cmd", userRequest.Command),
			zap.String("Status", userResponse.Status),
			zap.Duration("Duration", time.Now().Sub(startTime)),
		)
		bytes, _ := json.Marshal(userResponse)
		conn.Conn.Write(
			websocket.Message{
				IsNative: true,
				Body:     bytes,
			},
		)
		gw.model.Report.CountDataOut(len(bytes))
	}
	return nil
}

func (gw *APP) checkSystemKey(ctx iris.Context) {
	apiKey := ctx.Params().Get("apiKey")
	resp := new(rpc.Response)
	if apiKey != gw.systemKey {
		ctx.StatusCode(http.StatusUnauthorized)
		resp.Error(global.ErrAccess, []string{})
		ctx.JSON(resp)
		return
	}

	// Go to next handler
	ctx.Next()
}

func (gw *APP) PushPlaceActivity(ctx iris.Context) {
	placeID := ctx.Params().Get("placeID")
	activity := ctx.Params().GetIntDefault("placeActivity", 0)
	switch activity {
	case nested.PlaceActivityActionPostAdd:
		place := gw.model.Place.GetByID(placeID, nil)
		if place != nil {
			gw.pusher.InternalPlaceActivitySyncPush(place.GetMemberIDs(), placeID, activity)
		}
	}
}
