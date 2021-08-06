package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/log"
	"io"
	"net"
	"os"
)

type mailInfo struct {
	Sender     string   `json:"sender"`
	Domain     string   `json:"domain"`
	Recipients []string `json:"recipients"`
	Buffer     []byte   `json:"buffer"`
}

func main() {
	// --Configurations
	sender := flag.String("s", "", "Sender Address")
	domain := flag.String("d", "", "domain")
	flag.Parse()

	recipients := flag.Args()

	buf := new(bytes.Buffer)
	io.Copy(buf, os.Stdin)

	m := mailInfo{
		Sender:     *sender,
		Domain:     *domain,
		Recipients: recipients,
		Buffer:     buf.Bytes(),
	}
	log.Debug(fmt.Sprintf("mail-instances::Postfix wants to store email from %s for %s", m.Sender, m.Recipients))
	b, err := json.Marshal(m)
	if err != nil {
		log.Error(err.Error())
	}
	conn, err := net.Dial("tcp", "127.0.0.1:2300")
	if err != nil {
		log.Error(err.Error())
	}
	_, err = conn.Write(b)
	if err != nil {
		log.Error(err.Error())
	}
	defer conn.Close()
}
