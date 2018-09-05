package main

import (
    "encoding/csv"
    "fmt"
    "log"
    "net"
    "net/mail"
    "os"
    "strings"

    "git.ronaksoftware.com/nested/server/model"
    "gopkg.in/fzerorubigd/onion.v3"
)

var (
    _Config  *onion.Onion
    _Domains []string
    _Model   *nested.Manager
)

// Requests
const (
    REQ_GET = "get"
    REQ_PUT = "put"
)

// Responses
const (
    RES_UNAVAILABLE = "500"
    RES_ERROR       = "400"
    RES_SUCCESS     = "200"
)

func main() {
    _Config = readConfig()
    _Domains = _Config.GetStringSlice("DOMAIN")

    // Instantiate Nested Model Manager
    if n, err := nested.NewManager(
        _Config.GetString("INSTANCE_ID"),
        _Config.GetString("MONGO_DSN"),
        _Config.GetString("REDIS_DSN"),
        _Config.GetInt("DEBUG_LEVEL"),
    ); err != nil {
        log.Println("MAIL-MAP::Main::Nested Manager Error::", err.Error())
        os.Exit(1)
    } else {
        _Model = n
    }

    log.Println("Start Listening")
    listener, err := net.Listen("tcp", ":2374")
    if err != nil {
        log.Fatal(err.Error())
    }
    defer listener.Close()

    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        handleConn(conn)
    }

}

func handleConn(conn net.Conn) {
    defer conn.Close()

    r := csv.NewReader(conn)
    r.Comma = ' '
    record, err := r.Read()
    if err != nil {
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
        conn.Close()

    }
    if !(len(record) == 2) {
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
        return
    }
    cmd := strings.ToLower(record[0])
    email, err := mail.ParseAddress(record[1])
    if err != nil {
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
        return
    }

    switch cmd {
    case REQ_GET:
        Get(conn, strings.ToLower(email.Address))
    case REQ_PUT:
    default:
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
    }
}

func Get(conn net.Conn, email string) {
    emailParts := strings.Split(email, "@")
    if len(emailParts) != 2 {
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
        return
    }
    placeID := emailParts[0]
    domain := emailParts[1]

    domainExists := false
    for _, myDomain := range _Domains {
        if myDomain == domain {
            domainExists = true
        }
    }
    if !domainExists {
        fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
        return
    }

    place := _Model.Place.GetByID(placeID, nil)
    if place == nil || place.Privacy.Receptive != nested.PLACE_RECEPTIVE_EXTERNAL {
        fmt.Fprintln(conn, fmt.Sprintf("%s Unavailable", RES_UNAVAILABLE))
        return
    }

    fmt.Fprintln(conn, fmt.Sprintf("%s %s", RES_SUCCESS, email))

}
