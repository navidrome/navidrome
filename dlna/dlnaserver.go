package dlna

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anacrolix/dms/soap"
	"github.com/anacrolix/dms/ssdp"
	"github.com/anacrolix/dms/upnp"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

const (
	serverField       = "Linux/3.4 DLNADOC/1.50 UPnP/1.0 DMS/1.0"
	rootDescPath      = "/rootDesc.xml"
	resPath           = "/r/"
	serviceControlURL = "/ctl"
)

type DLNAServer struct {
	ds     model.DataStore
	broker events.Broker
	ssdp   SSDPServer
}

type SSDPServer struct {
	// The service SOAP handler keyed by service URN.
	services map[string]UPnPService

	Interfaces []net.Interface

	HTTPConn       net.Listener
	httpListenAddr string
	handler        http.Handler

	RootDeviceUUID string

	FriendlyName string

	// For waiting on the listener to close
	waitChan chan struct{}

	// Time interval between SSPD announces
	AnnounceInterval time.Duration
}

func New(ds model.DataStore, broker events.Broker) *DLNAServer {
	s := &DLNAServer{ds: ds, broker: broker, ssdp: SSDPServer{}}
	s.ssdp.Interfaces = listInterfaces()

	return s
}

// Run starts the server with the given address, and if specified, with TLS enabled.
func (s *DLNAServer) Run(ctx context.Context, addr string, port int, tlsCert string, tlsKey string) error {
	return nil
}

type UPnPService interface {
	Handle(action string, argsXML []byte, r *http.Request) (respArgs map[string]string, err error)
	Subscribe(callback []*url.URL, timeoutSeconds int) (sid string, actualTimeout int, err error)
	Unsubscribe(sid string) error
}

func (s *SSDPServer) startSSDP() {
	active := 0
	stopped := make(chan struct{})
	for _, intf := range s.Interfaces {
		active++
		go func(intf2 net.Interface) {
			defer func() {
				stopped <- struct{}{}
			}()
			s.ssdpInterface(intf2)
		}(intf)
	}
	for active > 0 {
		<-stopped
		active--
	}
}

// Run SSDP server on an interface.
func (s *SSDPServer) ssdpInterface(intf net.Interface) {
	// Figure out whether should an ip be announced
	ipfilterFn := func(ip net.IP) bool {
		listenaddr := s.HTTPConn.Addr().String()
		listenip := listenaddr[:strings.LastIndex(listenaddr, ":")]
		switch listenip {
		case "0.0.0.0":
			if strings.Contains(ip.String(), ":") {
				// Any IPv6 address should not be announced
				// because SSDP only listen on IPv4 multicast address
				return false
			}
			return true
		case "[::]":
			// In the @Serve() section, the default settings have been made to not listen on IPv6 addresses.
			// If actually still listening on [::], then allow to announce any address.
			return true
		default:
			if listenip == ip.String() {
				return true
			}
			return false
		}
	}

	// Figure out which HTTP location to advertise based on the interface IP.
	advertiseLocationFn := func(ip net.IP) string {
		url := url.URL{
			Scheme: "http",
			Host: (&net.TCPAddr{
				IP:   ip,
				Port: s.HTTPConn.Addr().(*net.TCPAddr).Port,
			}).String(),
			Path: rootDescPath,
		}
		return url.String()
	}

	_, err := intf.Addrs()
	if err != nil {
		panic(err)
	}
	log.Printf("Started SSDP on %v", intf.Name)

	// Note that the devices and services advertised here via SSDP should be
	// in agreement with the rootDesc XML descriptor that is defined above.
	ssdpServer := ssdp.Server{
		Interface: intf,
		Devices: []string{
			"urn:schemas-upnp-org:device:MediaServer:1"},
		Services: []string{
			"urn:schemas-upnp-org:service:ContentDirectory:1",
			"urn:schemas-upnp-org:service:ConnectionManager:1",
			"urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1"},
		IPFilter:       ipfilterFn,
		Location:       advertiseLocationFn,
		Server:         serverField,
		UUID:           s.RootDeviceUUID,
		NotifyInterval: s.AnnounceInterval,
	}

	// An interface with these flags should be valid for SSDP.
	const ssdpInterfaceFlags = net.FlagUp | net.FlagMulticast

	if err := ssdpServer.Init(); err != nil {
		if intf.Flags&ssdpInterfaceFlags != ssdpInterfaceFlags {
			// Didn't expect it to work anyway.
			return
		}
		if strings.Contains(err.Error(), "listen") {
			// OSX has a lot of dud interfaces. Failure to create a socket on
			// the interface are what we're expecting if the interface is no
			// good.
			return
		}
		log.Printf("Error creating ssdp server on %s: %s", intf.Name, err)
		return
	}
	defer ssdpServer.Close()
	log.Printf("Started SSDP on %v", intf.Name)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		if err := ssdpServer.Serve(); err != nil {
			log.Printf("%q: %q\n", intf.Name, err)
		}
	}()
	select {
	case <-s.waitChan:
		// Returning will close the server.
	case <-stopped:
	}
}

// Get all available active network interfaces.
func listInterfaces() []net.Interface {
	ifs, err := net.Interfaces()
	if err != nil {
		log.Println("list network interfaces: %v", err)
		return []net.Interface{}
	}

	var active []net.Interface
	for _, intf := range ifs {
		if isAppropriatelyConfigured(intf) {
			active = append(active, intf)
		}
	}
	return active
}
func isAppropriatelyConfigured(intf net.Interface) bool {
	return intf.Flags&net.FlagUp != 0 && intf.Flags&net.FlagMulticast != 0 && intf.MTU > 0
}

func didlLite(chardata string) string {
	return `<DIDL-Lite` +
		` xmlns:dc="http://purl.org/dc/elements/1.1/"` +
		` xmlns:upnp="urn:schemas-upnp-org:metadata-1-0/upnp/"` +
		` xmlns="urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/"` +
		` xmlns:dlna="urn:schemas-dlna-org:metadata-1-0/">` +
		chardata +
		`</DIDL-Lite>`
}

func mustMarshalXML(value interface{}) []byte {
	ret, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Panicf("mustMarshalXML failed to marshal %v: %s", value, err)
	}
	return ret
}

// Marshal SOAP response arguments into a response XML snippet.
func marshalSOAPResponse(sa upnp.SoapAction, args map[string]string) []byte {
	soapArgs := make([]soap.Arg, 0, len(args))
	for argName, value := range args {
		soapArgs = append(soapArgs, soap.Arg{
			XMLName: xml.Name{Local: argName},
			Value:   value,
		})
	}
	return []byte(fmt.Sprintf(`<u:%[1]sResponse xmlns:u="%[2]s">%[3]s</u:%[1]sResponse>`,
		sa.Action, sa.ServiceURN.String(), mustMarshalXML(soapArgs)))
}
