package ronak

import (
    "sync"
    "github.com/mailru/easygo/netpoll"
    "github.com/gobwas/ws"
    "net"
    "log"
    "time"
    "github.com/gobwas/pool/pbytes"
    "io"
)

// WebsocketServer
type WebsocketServer struct {
    sync.Mutex
    chLimit            chan bool
    poller             netpoll.Poller
    upgrade            ws.Upgrader
    lastConnID         uint64
    maxConns           uint64
    conns              map[uint64]*WebsocketConnection
    OnWebsocketMessage func(wsConn *WebsocketConnection, payload []byte)
    OnWebsocketClose   func(wsConn *WebsocketConnection, code ws.StatusCode, text string)
}

func NewWebsocketServer(maxConcurrency int) *WebsocketServer {
    s := new(WebsocketServer)
    s.conns = make(map[uint64]*WebsocketConnection)

    // Setup NetPoll
    if p, err := netpoll.New(nil); err != nil {
        _LOG.Fatal("NetPoll", err.Error())
    } else {
        s.poller = p
    }

    s.upgrade = ws.Upgrader{

    }
    s.chLimit = make(chan bool, maxConcurrency)
    return s
}

func (s *WebsocketServer) LimitUp() {
    s.chLimit <- true
}

func (s *WebsocketServer) LimitDown() {
    <-s.chLimit
}

func (s *WebsocketServer) AddConnection(conn net.Conn, connDesc *netpoll.Desc) *WebsocketConnection {
    _funcName := "WebsocketServer::AddConnection"
    s.Lock()
    s.lastConnID++
    wsConn := WebsocketConnection{
        ID:        s.lastConnID,
        conn:      conn,
        connDesc:  connDesc,
        server:    s,
        OnMessage: s.OnWebsocketMessage,
    }
    s.conns[wsConn.ID] = &wsConn
    _LOG.Debug(_funcName, "", "Total Connections:", len(s.conns))
    s.Unlock()
    return &wsConn
}

func (s *WebsocketServer) RemoveConnection(wcID uint64) {
    _funcName := "WebsocketServer::RemoveConnection"
    s.Lock()
    if wc, ok := s.conns[wcID]; ok {
        wc.conn.Close()
        delete(s.conns, wcID)
    }
    _LOG.Debug(_funcName, "", "Total Connections:", len(s.conns))
    s.Unlock()
}

func (s *WebsocketServer) GetConnection(wcID uint64) *WebsocketConnection {
    s.Lock()
    defer s.Unlock()
    if wc, ok := s.conns[wcID]; ok {
        return wc
    }
    return nil
}

// Run
// This function is a blocking call and will loop forever until Shutdown signal received.
// TODO:: (ehsan) implement Graceful Shutdown
func (s *WebsocketServer) Run() {
    _funcName := "WebsocketServer::Run"
    listener, err := net.Listen("tcp", ":8081")
    if err != nil {
        _LOG.Fatal(_funcName, err.Error())
    }

    // Start Infinite Loop to Accept new Connections
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }

        // Try to Upgrade Connection
        if _, err := ws.Upgrade(conn); err != nil {
            _LOG.Error(_funcName, err.Error(), "Websocket Upgrade")
            continue
        }

        // Register NetPoll event handler
        // The connDesc (Connection Describer) will be registered in an observer-list.
        if connDesc, err := netpoll.HandleReadOnce(conn); err != nil {
            _LOG.Error(_funcName, err.Error())

            // Close the connection and free resources (i.e. File Descriptor, ...)
            conn.Close()
        } else {
            wsConn := s.AddConnection(conn, connDesc)
            if err := s.poller.Start(connDesc, func(e netpoll.Event) {
                // TODO:: (ehsan) Block for ever ?!!
                s.LimitUp()
                go func() {
                    defer s.LimitDown()
                    wsConn.Receive()
                }()

            }); err != nil {
                log.Println(err.Error())
            }
        }
    }
}

// WebsocketConnection
type WebsocketConnection struct {
    sync.Mutex
    ID          uint64
    ConnToken   int64
    ReadTimeout time.Duration
    OnMessage   func(wc *WebsocketConnection, payload []byte)

    server   *WebsocketServer
    conn     net.Conn
    connDesc *netpoll.Desc
}

// Receive
// This function will be triggered when Poller triggers a READ event, and this routine
// reads everything in the buffer until either:
//  1. End of File
//  2. Read Timeout before next incoming frame
//  3. Exceeding the limit of
func (wc *WebsocketConnection) Receive() {
    _funcName := "WebsocketConnection::Receive"

    for {
        wc.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
        f, err := wc.readFrame(wc.conn)
        if err != nil {
            pbytes.Put(f.Payload)
            switch err {
            case io.EOF:
                wc.server.RemoveConnection(wc.ID)
            default:
                wc.server.poller.Resume(wc.connDesc)
            }
            return
        }

        if f.Header.Masked {
            ws.Cipher(f.Payload, f.Header.Mask, 0)
        }

        switch f.Header.OpCode {
        case ws.OpBinary, ws.OpText:
            wc.OnMessage(wc, f.Payload)
        case ws.OpClose:
            statusCode, reason := ws.ParseCloseFrameData(f.Payload)
            _LOG.Debug(_funcName, "Connection Closed by Client", statusCode, reason)
            pbytes.Put(f.Payload)
            wc.server.RemoveConnection(wc.ID)
            return
        case ws.OpPing:

        default:
            _LOG.Error(_funcName, "OpCode Not Caught", f.Header.OpCode, f.Header.Length)
        }
        pbytes.Put(f.Payload)
    }

    wc.server.poller.Resume(wc.connDesc)
    return
}

func (wc *WebsocketConnection) readFrame(r io.Reader) (f ws.Frame, err error) {
    f.Header, err = ws.ReadHeader(r)
    if err != nil {
        return
    }

    if f.Header.Length > 0 {
        // int(f.Header.Length) is safe here cause we have
        // checked it for overflow above in ReadHeader.
        f.Payload = pbytes.GetLen(int(f.Header.Length))
        _, err = io.ReadFull(r, f.Payload)
    }

    return
}

func (wc *WebsocketConnection) Send(payload []byte) {
    _funcName := "WebsocketConnection::Send"
    if err := ws.WriteFrame(wc.conn, ws.NewBinaryFrame(payload)); err != nil {
        _LOG.Error(_funcName, err.Error())
        wc.server.poller.Stop(wc.connDesc)
        wc.server.RemoveConnection(wc.ID)
    }
}
