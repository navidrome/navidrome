package dlna

import (
	"bytes"
	"context"
	"crypto/md5"
	"embed"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	dms_dlna "github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/dms/soap"
	"github.com/anacrolix/dms/ssdp"
	"github.com/anacrolix/dms/upnp"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

const (
	serverField       = "Linux/3.4 DLNADOC/1.50 UPnP/1.0 DMS/1.0"
	rootDescPath      = "/rootDesc.xml"
	resPath           = "/r/"
	serviceControlURL = "/ctl"
)

//go:embed static/*
var staticContent embed.FS

type DLNAServer struct {
	ds     model.DataStore
	broker events.Broker
	ssdp   SSDPServer
	ctx    context.Context
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
	ModelNumber  string

	// For waiting on the listener to close
	waitChan chan struct{}

	// Time interval between SSPD announces
	AnnounceInterval time.Duration
}

func New(ds model.DataStore, broker events.Broker) *DLNAServer {
	s := &DLNAServer{
		ds:     ds,
		broker: broker,
		ssdp: SSDPServer{
			AnnounceInterval: time.Duration(30) * time.Second,
			Interfaces:       listInterfaces(),
			FriendlyName:     "Navidrome",
			ModelNumber:      consts.Version,
			RootDeviceUUID:   makeDeviceUUID("Navidrome"),
			waitChan:         make(chan struct{}),
		},
	}

	s.ssdp.services = map[string]UPnPService{
		"ContentDirectory": &contentDirectoryService{
			DLNAServer: s,
		},
		"ConnectionManager": &connectionManagerService{
			DLNAServer: s,
		},
		"X_MS_MediaReceiverRegistrar": &mediaReceiverRegistrarService{
			DLNAServer: s,
		},
	}

	//setup dedicated HTTP server for UPNP
	r := http.NewServeMux()
	r.Handle(resPath, http.StripPrefix(resPath, http.HandlerFunc(s.ssdp.resourceHandler)))

	r.Handle("/static/", http.FileServer(http.FS(staticContent)))
	r.HandleFunc(rootDescPath, s.ssdp.rootDescHandler)
	r.HandleFunc(serviceControlURL, s.ssdp.serviceControlHandler)

	s.ssdp.handler = r

	return s
}

// Run starts the DLNA server (both SSDP and HTTP) with the given address
func (s *DLNAServer) Run(ctx context.Context, addr string, port int) (err error) {
	log.Warn("Starting DLNA Server")

	s.ctx = ctx
	if s.ssdp.HTTPConn == nil {
		network := "tcp4"
		if strings.Count(s.ssdp.httpListenAddr, ":") > 1 {
			network = "tcp"
		}
		s.ssdp.HTTPConn, err = net.Listen(network, s.ssdp.httpListenAddr)
		if err != nil {
			return
		}
	}
	go func() {
		s.ssdp.startSSDP()
	}()
	go func() {
		s.ssdp.serveHTTP()
	}()
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
	log.Info(fmt.Sprintf("Started SSDP on %v", intf.Name))

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
		log.Error(fmt.Sprintf("Error creating ssdp server on %s: %s", intf.Name), err)
		return
	}
	defer ssdpServer.Close()

	log.Info(fmt.Sprintf("Started SSDP on %v", intf.Name))
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		if err := ssdpServer.Serve(); err != nil {
			log.Error(fmt.Sprintf("Err %q", intf.Name), err)

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

// handler for all paths under `/r`
func (s *SSDPServer) resourceHandler(w http.ResponseWriter, r *http.Request) {
	remotePath := r.URL.Path

	localFile, _ := strings.CutPrefix(remotePath, "Music/Files/")
	localFilePath := path.Join(conf.Server.MusicFolder, localFile)

	log.Info(fmt.Sprintf("resource handler Executed with remote path: %s, localpath: %s", remotePath, localFilePath))

	fileStats, err := os.Stat(localFilePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(fileStats.Size(), 10))

	// add some DLNA specific headers
	if r.Header.Get("getContentFeatures.dlna.org") != "" {
		w.Header().Set("contentFeatures.dlna.org", dms_dlna.ContentFeatures{
			SupportRange: true,
		}.String())
	}
	w.Header().Set("transferMode.dlna.org", "Streaming")

	os.Open(localFilePath)
	fileHandle, err := os.Open(localFilePath)
	if err != nil {
		fmt.Printf("file streaming error: %+v\n", err)
		return
	}
	defer fileHandle.Close()

	http.ServeContent(w, r, remotePath, time.Now(), fileHandle)
}

// returns /rootDesc.xml templated
func (s *SSDPServer) rootDescHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := GetTemplate()

	buffer := new(bytes.Buffer)
	_ = tmpl.Execute(buffer, s)

	w.Header().Set("content-type", `text/xml; charset="utf-8"`)
	w.Header().Set("cache-control", "private, max-age=60")
	w.Header().Set("content-length", strconv.FormatInt(int64(buffer.Len()), 10))
	buffer.WriteTo(w)
}

// Handle a service control HTTP request.
func (s *SSDPServer) serviceControlHandler(w http.ResponseWriter, r *http.Request) {
	soapActionString := r.Header.Get("SOAPACTION")
	soapAction, err := upnp.ParseActionHTTPHeader(soapActionString)
	if err != nil {
		serveError(s, w, "Could not parse SOAPACTION header", err)
		return
	}
	var env soap.Envelope
	if err := xml.NewDecoder(r.Body).Decode(&env); err != nil {
		serveError(s, w, "Could not parse SOAP request body", err)
		return
	}

	w.Header().Set("Content-Type", `text/xml; charset="utf-8"`)
	w.Header().Set("Ext", "")
	soapRespXML, code := func() ([]byte, int) {
		respArgs, err := s.soapActionResponse(soapAction, env.Body.Action, r)
		if err != nil {
			fmt.Printf("Error invoking %v: %v", soapAction, err)
			upnpErr := upnp.ConvertError(err)
			return mustMarshalXML(soap.NewFault("UPnPError", upnpErr)), http.StatusInternalServerError
		}
		return marshalSOAPResponse(soapAction, respArgs), http.StatusOK
	}()
	bodyStr := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" standalone="yes"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>%s</s:Body></s:Envelope>`, soapRespXML)
	w.WriteHeader(code)
	if _, err := w.Write([]byte(bodyStr)); err != nil {
		fmt.Printf("Error writing response: %v", err)
	}
}

// Handle a SOAP request and return the response arguments or UPnP error.
func (s *SSDPServer) soapActionResponse(sa upnp.SoapAction, actionRequestXML []byte, r *http.Request) (map[string]string, error) {
	service, ok := s.services[sa.Type]
	if !ok {
		// TODO: What's the invalid service error?
		return nil, upnp.Errorf(upnp.InvalidActionErrorCode, "Invalid service: %s", sa.Type)
	}
	return service.Handle(sa.Action, actionRequestXML, r)
}

func (s *SSDPServer) serveHTTP() error {
	srv := &http.Server{
		Handler: s.handler,
	}
	err := srv.Serve(s.HTTPConn)
	select {
	case <-s.waitChan:
		return nil
	default:
		return err
	}
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
		log.Fatal(fmt.Sprintf("mustMarshalXML failed to marshal %v: %s $s", value, err))
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

func makeDeviceUUID(unique string) string {
	h := md5.New()
	if _, err := io.WriteString(h, unique); err != nil {
		log.Fatal(fmt.Sprintf("makeDeviceUUID write failed: %s", err))
	}
	buf := h.Sum(nil)
	return upnp.FormatUUID(buf)
}

// HTTP handler that sets headers.
func withHeader(name string, value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(name, value)
		next.ServeHTTP(w, r)
	})
}

// serveError returns an http.StatusInternalServerError and logs the error
func serveError(what interface{}, w http.ResponseWriter, text string, err error) {
	http.Error(w, text+".", http.StatusInternalServerError)
	log.Error(fmt.Sprintf("serveError: %s, %s, %s", what, text), err)
}

func GetTemplate() (tpl *template.Template, err error) {

	templateBytes := `<?xml version="1.0"?>
<root xmlns="urn:schemas-upnp-org:device-1-0"
      xmlns:dlna="urn:schemas-dlna-org:device-1-0"
      xmlns:sec="http://www.sec.co.kr/dlna">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <device>
    <deviceType>urn:schemas-upnp-org:device:MediaServer:1</deviceType>
    <friendlyName>{{.FriendlyName}}</friendlyName>
    <manufacturer>Navidrome</manufacturer>
    <manufacturerURL>https://www.navidrome.org/</manufacturerURL>
    <modelDescription>Navidrome</modelDescription>
    <modelName>Navidrome</modelName>
    <modelNumber>{{.ModelNumber}}</modelNumber>
    <modelURL>https://www.navidrome.org/</modelURL>
    <serialNumber>00000000</serialNumber>
    <UDN>{{.RootDeviceUUID}}</UDN>
    <dlna:X_DLNACAP/>
    <dlna:X_DLNADOC>DMS-1.50</dlna:X_DLNADOC>
    <dlna:X_DLNADOC>M-DMS-1.50</dlna:X_DLNADOC>
    <sec:ProductCap>smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec</sec:ProductCap>
    <sec:X_ProductCap>smi,DCM10,getMediaInfo.sec,getCaptionInfo.sec</sec:X_ProductCap>
    <serviceList>
      <service>
        <serviceType>urn:schemas-upnp-org:service:ContentDirectory:1</serviceType>
        <serviceId>urn:upnp-org:serviceId:ContentDirectory</serviceId>
        <SCPDURL>/static/ContentDirectory.xml</SCPDURL>
        <controlURL>/ctl</controlURL>
        <eventSubURL></eventSubURL>
      </service>
      <service>
        <serviceType>urn:schemas-upnp-org:service:ConnectionManager:1</serviceType>
        <serviceId>urn:upnp-org:serviceId:ConnectionManager</serviceId>
        <SCPDURL>/static/ConnectionManager.xml</SCPDURL>
        <controlURL>/ctl</controlURL>
        <eventSubURL></eventSubURL>
      </service>
      <service>
        <serviceType>urn:microsoft.com:service:X_MS_MediaReceiverRegistrar:1</serviceType>
        <serviceId>urn:microsoft.com:serviceId:X_MS_MediaReceiverRegistrar</serviceId>
        <SCPDURL>/static/X_MS_MediaReceiverRegistrar.xml</SCPDURL>
        <controlURL>/ctl</controlURL>
        <eventSubURL></eventSubURL>
      </service>
    </serviceList>
    <presentationURL>/</presentationURL>
  </device>
</root>`

	var templateString = string(templateBytes)

	tpl, err = template.New("rootDesc").Parse(templateString)
	if err != nil {
		return nil, fmt.Errorf("get template parse: %w", err)
	}

	return
}
