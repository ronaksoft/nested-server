package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"

	"git.ronaksoftware.com/nested/server/model"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/globalsign/mgo"
	"gopkg.in/fzerorubigd/onion.v3"
	"net"
	"net/mail"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	_Config *onion.Onion
)

// Requests
const (
	COLLECTION_PLACES = "places"
	REQ_GET           = "get"
	REQ_PUT           = "put"
)

// Responses
const (
	RES_UNAVAILABLE = "500"
	RES_ERROR       = "400"
	RES_SUCCESS     = "200"
)

type info struct {
	InstanceID   string
	SystemKey    string
	CyrusURL     string
	MongoSession *mgo.Session
	SmtpUser     string
	SmtpPass     string
}

var instanceInfo = make(map[string]info)

func main() {
	_Config = readConfig()
	fmt.Println("starting mail-map")
	cli, err := client.NewClient(client.DefaultDockerHost, "1.37", nil, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	ctx := context.Background()
	args := filters.NewArgs()
	args.Add("name", "gateway")
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: args})
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, container := range containers {
		env, _ := cli.ContainerInspect(ctx, container.ID)
		envs := make(map[string]string, len(env.Config.Env))
		for _, item := range env.Config.Env {
			parts := strings.Split(item, "=")
			envs[parts[0]] = parts[1]
		}
		session, _ := initMongo(envs["NST_MONGO_DSN"])
		instanceInfo[envs["NST_DOMAIN"]] = info{
			InstanceID:   envs["NST_INSTANCE_ID"],
			SystemKey:    envs["NST_FILE_SYSTEM_KEY"],
			CyrusURL:     envs["NST_CYRUS_URL"],
			MongoSession: session,
			SmtpUser:     envs["NST_SMTP_USER"],
			SmtpPass:     envs["NST_SMTP_PASS"],
		}
	}

	// set multiple domains in postfix virtual_domains
	f, err := os.OpenFile("/etc/postfix/virtual_domains", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println("virtual_domains", err)
	}
	defer f.Close()
	for key := range instanceInfo {
		if _, err = f.WriteString(key + "\n"); err != nil {
			fmt.Println("virtual_domains::WriteString", err)
		}
	}

	// opendkim configs
	t, err := os.OpenFile("/etc/opendkim/TrustedHosts", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println("TrustedHosts", err)
	}
	defer t.Close()
	for key := range instanceInfo {
		if _, err = t.WriteString(key + "\n"); err != nil {
			fmt.Println("TrustedHosts::WriteString", err)
		}
	}

	k, err := os.OpenFile("/etc/opendkim/KeyTable", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
	}
	defer k.Close()
	for key := range instanceInfo {
		if _, err = k.WriteString(fmt.Sprintf("default._domainkey.%s %s:default:/etc/opendkim/domainkeys/%s/default.private\n", key, key, key)); err != nil {
			fmt.Println("KeyTable::WriteString", err)
		}
	}

	s, err := os.OpenFile("/etc/opendkim/SigningTable", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println("SigningTable::", err)
	}
	defer s.Close()
	for key := range instanceInfo {
		if _, err = s.WriteString(fmt.Sprintf("*@%s default._domainkey.%s\n", key, key)); err != nil {
			fmt.Println("SigningTable::WriteString", err)
		}
	}

	// run opendkim
	_, err = exec.Command("opendkim", "-A").Output()
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println("mail-map::instanceInfo", instanceInfo)
	go runEvery(time.Minute*time.Duration(_Config.GetInt("WATCHDOG_INTERVAL")), watchdog)
	fmt.Println("mail-map::Start Listening tcp:2374")
	listener, err := net.Listen("tcp", ":2374")
	if err != nil {
		fmt.Println(err.Error())
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

func initMongo(mongoDSN string) (*mgo.Session, error) {
	// Initial MongoDB
	tlsConfig := new(tls.Config)
	tlsConfig.InsecureSkipVerify = true
	if dialInfo, err := mgo.ParseURL(mongoDSN); err != nil {
		fmt.Println("initMongo::MongoDB URL Parse Failed::", err.Error())
		return nil, err
	} else {
		dialInfo.Timeout = 5 * time.Second
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			if conn, err := tls.Dial("tcp", addr.String(), tlsConfig); err != nil {
				return conn, err
			} else {
				return conn, nil
			}
		}
		if mongoSession, err := mgo.DialWithInfo(dialInfo); err != nil {
			fmt.Println("initMongo::DialWithInfo Failed::", err.Error())
			if mongoSession, err = mgo.Dial(mongoDSN); err != nil {
				fmt.Println("initMongo::Dial Failed::", err.Error())
				return nil, err
			} else {
				fmt.Println("initMongo::MongoDB Connected")
				return mongoSession, nil
			}
		} else {
			fmt.Println("initMongo::MongoDB(TLS) Connected")
			return mongoSession, nil
		}
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	r := csv.NewReader(conn)
	r.Comma = ' '
	record, err := r.Read()
	fmt.Println("mail-map::incoming record to tcp:2374 from postfix", record)
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
	fmt.Println("==========email", email)
	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
		return
	}
	placeID := emailParts[0]
	domain := emailParts[1]

	domainExists := false
	for myDomain, _ := range instanceInfo {
		fmt.Println(myDomain)
		if myDomain == domain {
			domainExists = true
		}
	}
	if !domainExists {
		fmt.Println("Domain not exist", email)
		fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
		return
	}
	sess := instanceInfo[domain].MongoSession.Clone()
	defer sess.Close()
	DB := fmt.Sprintf("nested-%s", instanceInfo[domain].InstanceID)
	var place *nested.Place
	if err := sess.DB(DB).C(COLLECTION_PLACES).FindId(placeID).One(&place); err != nil {
		fmt.Fprintln(conn, fmt.Sprintf("%s COMMAND READ ERROR", RES_ERROR))
		fmt.Println("mail-map::COLLECTION_PLACES", err.Error())
		return
	}

	if place == nil || place.Privacy.Receptive != nested.PLACE_RECEPTIVE_EXTERNAL {
		fmt.Fprintln(conn, fmt.Sprintf("%s Unavailable", RES_UNAVAILABLE))
		return
	}

	fmt.Fprintln(conn, fmt.Sprintf("%s %s", RES_SUCCESS, email))
	fmt.Println("RES_SUCCESS", email)

}

func watchdog(t time.Time) {
	cli, err := client.NewClient(client.DefaultDockerHost, "1.37", nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	ctx := context.Background()
	args := filters.NewArgs()
	args.Add("name", "gateway")

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: args})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	domains := make([]string, 0, len(containers))
	if len(containers) != len(instanceInfo) {
		for _, container := range containers {
			env, _ := cli.ContainerInspect(ctx, container.ID)
			envs := make(map[string]string, len(env.Config.Env))
			for _, item := range env.Config.Env {
				parts := strings.Split(item, "=")
				envs[parts[0]] = parts[1]
			}
			domains = append(domains, envs["NST_DOMAIN"])
			if _, ok := instanceInfo[envs["NST_DOMAIN"]]; ok {
				continue
			} else {
				session, _ := initMongo(envs["NST_MONGO_DSN"])
				instanceInfo[envs["NST_DOMAIN"]] = info{
					InstanceID:   envs["NST_INSTANCE_ID"],
					SystemKey:    envs["NST_FILE_SYSTEM_KEY"],
					CyrusURL:     envs["NST_CYRUS_URL"],
					MongoSession: session,
				}
			}
		}
		for domainBefore := range instanceInfo {
			exist := false
			for _, domain := range domains {
				if domain == domainBefore {
					exist = true
					continue
				}
			}
			if exist == false {
				delete(instanceInfo, domainBefore)
			}
		}
		os.Remove("/etc/postfix/virtual_domains")
		f, err := os.OpenFile("/etc/postfix/virtual_domains", os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println(err)
		}
		defer f.Close()
		for key := range instanceInfo {
			if _, err = f.WriteString(key + "\n"); err != nil {
				fmt.Println(err)
			}
		}
		// opendkim configs
		t, err := os.OpenFile("/etc/opendkim/TrustedHosts", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println("TrustedHosts", err)
		}
		defer t.Close()
		for key := range instanceInfo {
			if _, err = t.WriteString(key + "\n"); err != nil {
				fmt.Println("TrustedHosts::WriteString", err)
			}
		}

		k, err := os.OpenFile("/etc/opendkim/KeyTable", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println(err)
		}
		defer k.Close()
		for key := range instanceInfo {
			if _, err = k.WriteString(fmt.Sprintf("default._domainkey.%s %s:default:/etc/opendkim/domainkeys/%s/default.private\n", key, key, key)); err != nil {
				fmt.Println("KeyTable::WriteString", err)
			}
		}

		s, err := os.OpenFile("/etc/opendkim/SigningTable", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println("SigningTable::", err)
		}
		defer s.Close()
		for key := range instanceInfo { //otherdomain.com default._domainkey.otherdomain.com
			if _, err = s.WriteString(fmt.Sprintf("*@%s default._domainkey.%s\n", key, key)); err != nil {
				fmt.Println("SigningTable::WriteString", err)
			}
		}
	}
}

func runEvery(t time.Duration, f func(time.Time)) {
	for x := range time.Tick(t) {
		f(x)
	}
}
