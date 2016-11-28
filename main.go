package main

import (
	"flag"
	"fmt"
	"io"
        "net"
	"net/http"
        "net/url"
        "crypto/tls"
	"os"
	"strings"
        "log"

	"golang.org/x/net/websocket"
)

var (
	origin  string
	headers string
	version int
        resolve_as string
)

func init() {
	help = fmt.Sprintf(help, VERSION)
	flag.StringVar(&origin, "o", "http://0.0.0.0/", "websocket origin")
	flag.StringVar(&headers, "H", "", "a comma separated list of http headers")
	flag.IntVar(&version, "v", websocket.ProtocolVersionHybi13, "websocket version")
        flag.StringVar(&resolve_as, "r", "", "resolve the host as a specified ip, or ip:port")
	flag.Parse()
        log.SetFlags(log.LstdFlags|log.Lshortfile)
}

const VERSION = "0.1"

var help = `ws - %s

Usage:
	ws [options] <url>

Use "ws --help" for help.
`

func parseHeaders(headers string) http.Header {
	h := http.Header{}
	for _, header := range strings.Split(headers, ",") {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			continue
		}
		h.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return h
}
func make_ip_port(u *url.URL) string{
    if u==nil {return ""}
    port := "80"
    if u.Scheme == "wss" {
        port = "443"
    }
    if _,_,err := net.SplitHostPort(u.Host); err!=nil { // not found a port
        return net.JoinHostPort(u.Host,port)
    }
    return u.Host
}
func main() {

        var (
            client net.Conn
            ws *websocket.Conn
            err error
        )

	target := flag.Arg(0)
	if target == "" {
		log.Fatal(help)
	}
	config, err := websocket.NewConfig(target, origin)
	if err != nil {
		log.Fatal("%s\n", err)
	}
	if headers != "" {
		config.Header = parseHeaders(headers)
	}
	config.Version = version

        // i dont use the default DialConfig, or Dial, in order to control more and fetch more infomation
        switch config.Location.Scheme {
        case "ws":
            if resolve_as == "" {
                client,err = net.Dial("tcp", make_ip_port(config.Location) )
            } else {
                u,err := url.Parse(fmt.Sprintf("ws://%s",resolve_as))
                if err != nil {
                    log.Fatal(err)
                }
                client,err = net.Dial("tcp",make_ip_port(u))
            }

        case "wss":
            config.TlsConfig = &tls.Config{ServerName: config.Location.Host} // override by myself
            if resolve_as == "" {
                client,err = tls.Dial("tcp",make_ip_port(config.Location), config.TlsConfig)
            } else {
                u,err := url.Parse(fmt.Sprintf("wss://%s",resolve_as))
                if err != nil {
                    log.Fatal(err)
                }
                client,err = tls.Dial("tcp",make_ip_port(u), config.TlsConfig)
            }
        default:
            log.Fatal("invalid scheme. should be ws:// or wss://\n")
        }
        if err != nil {
            log.Fatal(err)
        }
        ws, err = websocket.NewClient(config, client)
        if err!= nil {
            client.Close()
            log.Fatal("Error dialing %s: %v\n", target, err)
        }
        fmt.Printf("Connected to %v\n", client.RemoteAddr() )
        errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	go cp(os.Stdout, ws)
	go cp(ws, os.Stdin)
	<-errc
}
