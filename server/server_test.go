package server

import (
	"../actions"
	"../proxy"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

type ServerTestSuite struct {
	BaseUrl        string
	serviceName    string
	suite.Suite
}

func (s *ServerTestSuite) SetupTest() {
}

func TestServerUnitTestSuite(t *testing.T) {
	s := new(ServerTestSuite)
	s.serviceName = "my-fancy-service"

	logPrintfOrig := logPrintf
	defer func() { logPrintf = logPrintfOrig }()
	logPrintf = func(format string, v ...interface{}) {}

	os.Setenv("SKIP_ADDRESS_VALIDATION", "false")

	suite.Run(t, s)
}

// Test1Handler

func (s *ServerTestSuite) Test_Test1Handler_ReturnsStatus200() {
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/test", nil)

	srv := serve{}
	srv.Test1Handler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

// Test2Handler

func (s *ServerTestSuite) Test_Test2Handler_ReturnsStatus200() {
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", "/v2/test", nil)

		srv := serve{}
		srv.Test2Handler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

// PingHandler

func (s *ServerTestSuite) Test_PingHandler_ReturnsStatus200() {
	rw := getResponseWriterMock()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/ping", nil)

	srv := serve{}
	srv.PingHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

// ReconfigureHandler

func (s *ServerTestSuite) Test_ReconfigureHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reconfigure?serviceName=my-service", nil)

	srv := serve{}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesErrorHeader_WhenReconfigureDistributeIsTrueAndError() {
	serve := serve{}
	serve.port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := sendDistributeRequests
	defer func() { sendDistributeRequests = sendDistributeRequestsOrig }()
	sendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, fmt.Errorf("This is an error")
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_WritesStatusOK_WhenReconfigureDistributeIsTrue() {
	serve := serve{}
	serve.port = "1234"
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&distribute=true&servicePath=/demo"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	sendDistributeRequestsOrig := sendDistributeRequests
	defer func() { sendDistributeRequests = sendDistributeRequestsOrig }()
	sendDistributeRequests = func(req *http.Request, port, serviceName string) (status int, err error) {
		return 0, nil
	}

	serve.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenServiceNameQueryIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus409_WhenModeIsHttpAndServicePathAndServiceDomainAreEmpty() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 409)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus200_WhenReqModeIsTcp() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				return nil
			},
		}
	}

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 200)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndSrcPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&port=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenReqModeIsTcpAndPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=redis&srcPort=6379&reqMode=tcp"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenModeIsServiceAndPortIsNotPresent() {
	url := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&serviceColor=orange&servicePath=/demo&serviceDomain=my-domain.com"
	req, _ := http.NewRequest("GET", url, nil)
	rw := getResponseWriterMock()

	srv := serve{mode: "service"}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus400_WhenModeIsSwarmAndPortIsNotPresent() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&serviceColor=orange&servicePath=/demo&serviceDomain=my-domain.com"
	req, _ := http.NewRequest("GET", addr, nil)
	rw := getResponseWriterMock()

	srv := serve{mode: "swARM"}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesReconfigureExecute() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath.1=/demo&port.1=1234"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				invoked = true
				return nil
			},
		}
	}

	srv := serve{mode: "swarm"}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_DoesNotInvokeReconfigureExecute_WhenDistributeIsTrue() {
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	invoked := false
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				invoked = true
				return nil
			},
		}
	}

	srv := serve{mode: "swarm"}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.False(invoked)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_ReturnsStatus500_WhenReconfigureExecuteFails() {
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				return fmt.Errorf("This is an error")
			},
		}
	}
	rw := getResponseWriterMock()
	addr := "/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234"
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{}
	srv.ReconfigureHandler(rw, req)

	rw.AssertCalled(s.T(), "WriteHeader", 500)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCert_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{
		mode: "swarm",
		cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-service", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesPutCertWithDomainName_WhenServiceCertIsPresent() {
	actualCertName := ""
	expectedCert := "my-cert with new line \\n"
	actualCert := ""
	cert := CertMock{
		PutCertMock: func(certName string, certContent []byte) (string, error) {
			actualCertName = certName
			actualCert = string(certContent[:])
			return "", nil
		},
	}
	addr := fmt.Sprintf("/v1/docker-flow-proxy/reconfigure?serviceName=my-service&servicePath=/demo&port=1234&serviceDomain=my-domain.com&serviceCert=%s", expectedCert)
	req, _ := http.NewRequest("GET", addr, nil)

	srv := serve{
		mode: "swarm",
		cert: cert,
	}
	srv.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal("my-domain.com", actualCertName)
	s.Equal(strings.Replace(expectedCert, "\\n", "\n", -1), actualCert)
}

func (s *ServerTestSuite) Test_ReconfigureHandler_InvokesReconfigureExecute_WhenConsulTemplatePathIsPresent() {
	sd := proxy.ServiceDest{
		ServicePath:   []string{},
		ReqMode:       "http",
		ServiceDomain: []string{},
	}
	pathFe := "/path/to/consul/fe/template"
	pathBe := "/path/to/consul/be/template"
	var actualBase actions.BaseReconfigure
	expectedBase := actions.BaseReconfigure{
		ConsulAddresses: []string{"http://my-consul.com"},
	}
	var actualService proxy.Service
	expectedService := proxy.Service{
		ServiceName:          "my-service",
		ConsulTemplateFePath: pathFe,
		ConsulTemplateBePath: pathBe,
		ServiceDest:          []proxy.ServiceDest{sd},
	}
	newReconfigureOrig := actions.NewReconfigure
	defer func() { actions.NewReconfigure = newReconfigureOrig }()
	invoked := false
	actions.NewReconfigure = func(baseData actions.BaseReconfigure, serviceData proxy.Service, mode string) actions.Reconfigurable {
		actualBase = baseData
		actualService = serviceData
		return ReconfigureMock{
			ExecuteMock: func(reloadAfter bool) error {
				invoked = true
				return nil
			},
		}
	}
	serverImpl := serve{
		consulAddresses: []string{"http://my-consul.com"},
	}
	addr := fmt.Sprintf(
		"/v1/docker-flow-proxy/reconfigure?serviceName=my-service&consulTemplateFePath=%s&consulTemplateBePath=%s",
		pathFe,
		pathBe,
	)
	req, _ := http.NewRequest("GET", addr, nil)

	serverImpl.ReconfigureHandler(getResponseWriterMock(), req)

	s.Equal(expectedBase, actualBase)
	s.Equal(expectedService, actualService)
	s.True(invoked)
}

// ReloadHandler

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsStatus200() {
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	for ver := 1; ver <= 2; ver++ {
		rw := getResponseWriterMock()
		req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

		srv := serve{}
		srv.ReloadHandler(rw, req)

		rw.AssertCalled(s.T(), "WriteHeader", 200)
	}
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReload() {
	invoked := false
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			invoked = true
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)

	srv := serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(invoked)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithRecreateParam() {
	actualRecreate := false
	clusterConfigCalled := false
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			actualRecreate = recreate
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			clusterConfigCalled = true
			return nil
		},
	})()

	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?recreate=true", nil)

	srv := serve{}
	srv.listenerAddress = "my-listener"
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.True(actualRecreate)
	s.False(clusterConfigCalled)
}

func (s *ServerTestSuite) Test_ReloadHandler_InvokesReloadWithFromListenerParam() {
	actualListenerAddr := ""
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	defer MockFetch(FetchMock{
		ReloadClusterConfigMock: func(listenerAddr string) error {
			actualListenerAddr = listenerAddr
			return nil
		},
	})()
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload?fromListener=true", nil)

	srv := serve{}
	srv.listenerAddress = "my-listener"
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal(srv.listenerAddress, actualListenerAddr)
}

func (s *ServerTestSuite) Test_ReloadHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()

	srv := serve{}
	srv.ReloadHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_ReloadHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status: "OK",
	})
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/reload", nil)
	defer MockReload(ReloadMock{
		ExecuteMock: func(recreate bool) error {
			return nil
		},
	})()
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.ReloadHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

// Remove Handler

func (s *ServerTestSuite) Test_RemoveHandler_SetsContentTypeToJSON() {
	var actual string
	httpWriterSetContentType = func(w http.ResponseWriter, value string) {
		actual = value
	}
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)

	srv := serve{}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal("application/json", actual)
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsJSON() {
	expected, _ := json.Marshal(Response{
		Status:      "OK",
		ServiceName: "my-service",
	})
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "Write", []byte(expected))
}

func (s *ServerTestSuite) Test_RemoveHandler_ReturnsStatus400_WhenServiceNameIsNotPresent() {
	req, _ := http.NewRequest("GET", "/v1/docker-flow-proxy/remove", nil)
	respWriterMock := getResponseWriterMock()

	srv := serve{}
	srv.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 400)
}

func (s *ServerTestSuite) Test_RemoveHandler_InvokesRemoveExecute() {
	mockObj := getRemoveMock("")
	aclName := "my-acl"
	var actual actions.Remove
	expected := actions.Remove{
		ServiceName:     s.serviceName,
		TemplatesPath:   "",
		ConfigsPath:     "",
		ConsulAddresses: []string{"http://1.2.3.4:1234"},
		InstanceName:    "proxy-test-instance",
		AclName:         aclName,
	}
	actions.NewRemove = func(
		serviceName, aclName, configsPath, templatesPath string,
		consulAddresses []string,
		instanceName, mode string,
	) actions.Removable {
		actual = actions.Remove{
			ServiceName:     serviceName,
			AclName:         aclName,
			TemplatesPath:   templatesPath,
			ConfigsPath:     configsPath,
			ConsulAddresses: consulAddresses,
			InstanceName:    instanceName,
			Mode:            mode,
		}
		return mockObj
	}
	url := fmt.Sprintf("/v1/docker-flow-proxy/remove?serviceName=%s&aclName=%s", s.serviceName, aclName)
	req, _ := http.NewRequest("GET", url, nil)

	srv := serve{
		consulAddresses: expected.ConsulAddresses,
		serviceName:     expected.InstanceName,
	}
	srv.RemoveHandler(getResponseWriterMock(), req)

	s.Equal(expected, actual)
	mockObj.AssertCalled(s.T(), "Execute", []string{})
}

func (s *ServerTestSuite) Test_RemoveHandler_WritesErrorHeader_WhenRemoveDistributeIsTrueAndItFails() {
	addr := "/v1/docker-flow-proxy/remove?serviceName=my-service&distribute=true"
	req, _ := http.NewRequest("GET", addr, nil)
	respWriterMock := getResponseWriterMock()

	serve := serve{}
	serve.RemoveHandler(respWriterMock, req)

	respWriterMock.AssertCalled(s.T(), "WriteHeader", 500)
}

// GetServiceFromUrl

func (s *ServerTestSuite) Test_GetServiceFromUrl_ReturnsProxyService() {
	expected := proxy.Service{
		AclName:               "aclName",
		AddReqHeader:          []string{"add-header-1", "add-header-2"},
		AddResHeader:          []string{"add-header-1", "add-header-2"},
		ConnectionMode:        "my-connection-mode",
		ConsulTemplateFePath:  "consulTemplateFePath",
		ConsulTemplateBePath:  "consulTemplateBePath",
		DelReqHeader:          []string{"add-header-1", "add-header-2"},
		DelResHeader:          []string{"add-header-1", "add-header-2"},
		Distribute:            true,
		HttpsOnly:             true,
		HttpsPort:             1234,
		OutboundHostname:      "outboundHostname",
		PathType:              "pathType",
		RedirectWhenHttpProto: true,
		ReqPathReplace:        "reqPathReplace",
		ReqPathSearch:         "reqPathSearch",
		ServiceCert:           "serviceCert",
		ServiceColor:          "serviceColor",
		ServiceDest: []proxy.ServiceDest{{
			ServiceDomain: []string{"domain1", "domain2"},
			ServicePath:   []string{"/"},
			Port:          "1234",
			ReqMode:       "reqMode",
		}},
		ServiceDomain:         []string{"domain1", "domain2"},
		ServiceDomainMatchAll: true,
		ServiceName:           "serviceName",
		SetReqHeader:          []string{"set-header-1", "set-header-2"},
		SetResHeader:          []string{"set-header-1", "set-header-2"},
		SslVerifyNone:         true,
		TemplateBePath:        "templateBePath",
		TemplateFePath:        "templateFePath",
		TimeoutServer:         "timeoutServer",
		TimeoutTunnel:         "timeoutTunnel",
		XForwardedProto:       true,
		Users: []proxy.User{{Username: "user1", Password: "pass1", PassEncrypted: true},
			{Username: "user2", Password: "pass2", PassEncrypted: true}},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&users=%s&usersPassEncrypted=%t&aclName=%s&serviceColor=%s&serviceCert=%s&outboundHostname=%s&consulTemplateFePath=%s&consulTemplateBePath=%s&pathType=%s&reqPathSearch=%s&reqPathReplace=%s&templateFePath=%s&templateBePath=%s&timeoutServer=%s&timeoutTunnel=%s&reqMode=%s&httpsOnly=%t&isDefaultBackend=%t&xForwardedProto=%t&redirectWhenHttpProto=%t&httpsPort=%d&serviceDomain=%s&distribute=%t&sslVerifyNone=%t&serviceDomainMatchAll=%t&addReqHeader=%s&addResHeader=%s&setReqHeader=%s&setResHeader=%s&delReqHeader=%s&delResHeader=%s&servicePath=/&port=1234&connectionMode=%s",
		s.BaseUrl,
		expected.ServiceName,
		"user1:pass1,user2:pass2",
		true,
		expected.AclName,
		expected.ServiceColor,
		expected.ServiceCert,
		expected.OutboundHostname,
		expected.ConsulTemplateFePath,
		expected.ConsulTemplateBePath,
		expected.PathType,
		expected.ReqPathSearch,
		expected.ReqPathReplace,
		expected.TemplateFePath,
		expected.TemplateBePath,
		expected.TimeoutServer,
		expected.TimeoutTunnel,
		expected.ServiceDest[0].ReqMode,
		expected.HttpsOnly,
		expected.IsDefaultBackend,
		expected.XForwardedProto,
		expected.RedirectWhenHttpProto,
		expected.HttpsPort,
		strings.Join(expected.ServiceDomain, ","),
		expected.Distribute,
		expected.SslVerifyNone,
		expected.ServiceDomainMatchAll,
		strings.Join(expected.AddReqHeader, ","),
		strings.Join(expected.AddResHeader, ","),
		strings.Join(expected.SetReqHeader, ","),
		strings.Join(expected.SetResHeader, ","),
		strings.Join(expected.DelReqHeader, ","),
		strings.Join(expected.DelResHeader, ","),
		expected.ConnectionMode,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal(expected, *actual)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_DefaultsReqModeToHttp() {
	req, _ := http.NewRequest("GET", s.BaseUrl+"?servicePath=/my-path", nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal("http", actual.ServiceDest[0].ReqMode)
}

func (s *ServerTestSuite) Test_GetServiceFromUrl_SetsServicePathToSlash_WhenDomainIsPresent() {
	expected := proxy.Service{
		ServiceDomain: []string{"domain1", "domain2"},
		ServiceName:   "serviceName",
		ServiceDest: []proxy.ServiceDest{
			{
				ServiceDomain: []string{"domain1", "domain2"},
				ServicePath: []string{"/"},
				Port: "1234",
				ReqMode: "http",
			},
		},
	}
	addr := fmt.Sprintf(
		"%s?serviceName=%s&serviceDomain=%s&port=%s",
		s.BaseUrl,
		expected.ServiceName,
		strings.Join(expected.ServiceDomain, ","),
		expected.ServiceDest[0].Port,
	)
	req, _ := http.NewRequest("GET", addr, nil)
	srv := serve{}

	actual := srv.GetServiceFromUrl(req)

	s.Equal(expected, *actual)
}

// GetServicesFromEnvVars

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsServices() {
	service := proxy.Service{
		AclName:               "my-AclName",
		AddReqHeader:          []string{"add-header-1", "add-header-2"},
		AddResHeader:          []string{"add-header-1", "add-header-2"},
		ConnectionMode:        "my-connection-mode",
		ConsulTemplateBePath:  "my-ConsulTemplateBePath",
		ConsulTemplateFePath:  "my-ConsulTemplateFePath",
		DelReqHeader:          []string{"del-header-1", "del-header-2"},
		DelResHeader:          []string{"del-header-1", "del-header-2"},
		Distribute:            true,
		HttpsOnly:             true,
		HttpsPort:             1234,
		IsDefaultBackend:      true,
		OutboundHostname:      "my-OutboundHostname",
		PathType:              "my-PathType",
		RedirectWhenHttpProto: true,
		ReqPathReplace:        "my-ReqPathReplace",
		ReqPathSearch:         "my-ReqPathSearch",
		ServiceCert:           "my-ServiceCert",
		ServiceDomain:         []string{"my-domain-1.com", "my-domain-2.com"},
		ServiceDomainMatchAll: true,
		ServiceName:           "my-ServiceName",
		SetReqHeader:          []string{"set-header-1", "set-header-2"},
		SetResHeader:          []string{"set-header-1", "set-header-2"},
		SslVerifyNone:         true,
		TemplateBePath:        "my-TemplateBePath",
		TemplateFePath:        "my-TemplateFePath",
		TimeoutServer:         "my-TimeoutServer",
		TimeoutTunnel:         "my-TimeoutTunnel",
		XForwardedProto:       true,
		ServiceDest: []proxy.ServiceDest{
			{Port: "1111", ServicePath: []string{"my-path-11", "my-path-12"}, SrcPort: 1112, ReqMode: "my-ReqMode"},
		},
	}
	os.Setenv("DFP_SERVICE_ACL_NAME", service.AclName)
	os.Setenv("DFP_SERVICE_ADD_REQ_HEADER", strings.Join(service.AddReqHeader, ","))
	os.Setenv("DFP_SERVICE_ADD_RES_HEADER", strings.Join(service.AddResHeader, ","))
	os.Setenv("DFP_SERVICE_CONNECTION_MODE", service.ConnectionMode)
	os.Setenv("DFP_SERVICE_CONSUL_TEMPLATE_FE_PATH", service.ConsulTemplateFePath)
	os.Setenv("DFP_SERVICE_CONSUL_TEMPLATE_BE_PATH", service.ConsulTemplateBePath)
	os.Setenv("DFP_SERVICE_DEL_REQ_HEADER", strings.Join(service.DelReqHeader, ","))
	os.Setenv("DFP_SERVICE_DEL_RES_HEADER", strings.Join(service.DelResHeader, ","))
	os.Setenv("DFP_SERVICE_DISTRIBUTE", strconv.FormatBool(service.Distribute))
	os.Setenv("DFP_SERVICE_HTTPS_ONLY", strconv.FormatBool(service.HttpsOnly))
	os.Setenv("DFP_SERVICE_HTTPS_PORT", strconv.Itoa(service.HttpsPort))
	os.Setenv("DFP_SERVICE_IS_DEFAULT_BACKEND", strconv.FormatBool(service.IsDefaultBackend))
	os.Setenv("DFP_SERVICE_OUTBOUND_HOSTNAME", service.OutboundHostname)
	os.Setenv("DFP_SERVICE_PATH_TYPE", service.PathType)
	os.Setenv("DFP_SERVICE_REDIRECT_WHEN_HTTP_PROTO", strconv.FormatBool(service.RedirectWhenHttpProto))
	os.Setenv("DFP_SERVICE_REQ_MODE", service.ServiceDest[0].ReqMode)
	os.Setenv("DFP_SERVICE_REQ_PATH_REPLACE", service.ReqPathReplace)
	os.Setenv("DFP_SERVICE_REQ_PATH_SEARCH", service.ReqPathSearch)
	os.Setenv("DFP_SERVICE_SERVICE_CERT", service.ServiceCert)
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN", strings.Join(service.ServiceDomain, ","))
	os.Setenv("DFP_SERVICE_SERVICE_DOMAIN_MATCH_ALL", strconv.FormatBool(service.ServiceDomainMatchAll))
	os.Setenv("DFP_SERVICE_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_SSL_VERIFY_NONE", strconv.FormatBool(service.SslVerifyNone))
	os.Setenv("DFP_SERVICE_TEMPLATE_BE_PATH", service.TemplateBePath)
	os.Setenv("DFP_SERVICE_TEMPLATE_FE_PATH", service.TemplateFePath)
	os.Setenv("DFP_SERVICE_TIMEOUT_SERVER", service.TimeoutServer)
	os.Setenv("DFP_SERVICE_TIMEOUT_TUNNEL", service.TimeoutTunnel)
	os.Setenv("DFP_SERVICE_X_FORWARDED_PROTO", strconv.FormatBool(service.XForwardedProto))
	os.Setenv("DFP_SERVICE_PORT", service.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_SERVICE_PATH", strings.Join(service.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SET_REQ_HEADER", strings.Join(service.SetReqHeader, ","))
	os.Setenv("DFP_SERVICE_SET_RES_HEADER", strings.Join(service.SetResHeader, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT", strconv.Itoa(service.ServiceDest[0].SrcPort))

	defer func() {
		os.Unsetenv("DFP_SERVICE_ACL_NAME")
		os.Unsetenv("DFP_SERVICE_ADD_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_ADD_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_CONNECTION_MODE")
		os.Unsetenv("DFP_SERVICE_CONSUL_TEMPLATE_BE_PATH")
		os.Unsetenv("DFP_SERVICE_CONSUL_TEMPLATE_FE_PATH")
		os.Unsetenv("DFP_SERVICE_DEL_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_DEL_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_DISTRIBUTE")
		os.Unsetenv("DFP_SERVICE_HTTPS_ONLY")
		os.Unsetenv("DFP_SERVICE_HTTPS_PORT")
		os.Unsetenv("DFP_SERVICE_IS_DEFAULT_BACKEND")
		os.Unsetenv("DFP_SERVICE_OUTBOUND_HOSTNAME")
		os.Unsetenv("DFP_SERVICE_PATH_TYPE")
		os.Unsetenv("DFP_SERVICE_PORT")
		os.Unsetenv("DFP_SERVICE_REDIRECT_WHEN_HTTP_PROTO")
		os.Unsetenv("DFP_SERVICE_REQ_MODE")
		os.Unsetenv("DFP_SERVICE_REQ_PATH_REPLACE")
		os.Unsetenv("DFP_SERVICE_REQ_PATH_SEARCH")
		os.Unsetenv("DFP_SERVICE_SERVICE_CERT")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN")
		os.Unsetenv("DFP_SERVICE_SERVICE_DOMAIN_MATCH_ALL")
		os.Unsetenv("DFP_SERVICE_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH")
		os.Unsetenv("DFP_SERVICE_SET_REQ_HEADER")
		os.Unsetenv("DFP_SERVICE_SET_RES_HEADER")
		os.Unsetenv("DFP_SERVICE_SRC_PORT")
		os.Unsetenv("DFP_SERVICE_SSL_VERIFY_NONE")
		os.Unsetenv("DFP_SERVICE_TEMPLATE_BE_PATH")
		os.Unsetenv("DFP_SERVICE_TEMPLATE_FE_PATH")
		os.Unsetenv("DFP_SERVICE_TIMEOUT_SERVER")
		os.Unsetenv("DFP_SERVICE_TIMEOUT_TUNNEL")
		os.Unsetenv("DFP_SERVICE_X_FORWARDED_PROTO")
	}()
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsServicesWithIndexedData() {
	service := proxy.Service{
		ServiceName: "my-ServiceName",
		ServiceDest: []proxy.ServiceDest{
			{Port: "1111", ServicePath: []string{"my-path-11", "my-path-12"}, SrcPort: 1112},
			{Port: "2221", ServicePath: []string{"my-path-21", "my-path-22"}, SrcPort: 2222},
		},
	}
	os.Setenv("DFP_SERVICE_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_PORT_1", service.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_SERVICE_PATH_1", strings.Join(service.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT_1", strconv.Itoa(service.ServiceDest[0].SrcPort))
	os.Setenv("DFP_SERVICE_PORT_2", service.ServiceDest[1].Port)
	os.Setenv("DFP_SERVICE_SERVICE_PATH_2", strings.Join(service.ServiceDest[1].ServicePath, ","))
	os.Setenv("DFP_SERVICE_SRC_PORT_2", strconv.Itoa(service.ServiceDest[1].SrcPort))

	defer func() {
		os.Unsetenv("DFP_SERVICE_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_PORT_1")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_1")
		os.Unsetenv("DFP_SERVICE_SRC_PORT_1")
		os.Unsetenv("DFP_SERVICE_PORT_2")
		os.Unsetenv("DFP_SERVICE_SERVICE_PATH_2")
		os.Unsetenv("DFP_SERVICE_SRC_PORT_2")
	}()
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	service.ServiceDest[0].ReqMode = "http"
	service.ServiceDest[1].ReqMode = "http"
	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsEmptyIfServiceNameIsNotSet() {
	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 0)
}

func (s *ServerTestSuite) Test_GetServicesFromEnvVars_ReturnsMultipleServices() {
	service := proxy.Service{
		ServiceName: "my-ServiceName",
		ServiceDest: []proxy.ServiceDest{
			{Port: "1111", ServicePath: []string{"my-path-11", "my-path-12"}, SrcPort: 1112, ReqMode: "http"},
		},
	}
	os.Setenv("DFP_SERVICE_1_SERVICE_NAME", service.ServiceName)
	os.Setenv("DFP_SERVICE_1_PORT", service.ServiceDest[0].Port)
	os.Setenv("DFP_SERVICE_1_SERVICE_PATH", strings.Join(service.ServiceDest[0].ServicePath, ","))
	os.Setenv("DFP_SERVICE_1_SRC_PORT", strconv.Itoa(service.ServiceDest[0].SrcPort))
	defer func() {
		os.Unsetenv("DFP_SERVICE_1_SERVICE_NAME")
		os.Unsetenv("DFP_SERVICE_1_PORT")
		os.Unsetenv("DFP_SERVICE_1_SERVICE_PATH")
		os.Unsetenv("DFP_SERVICE_1_SRC_PORT")
	}()

	srv := serve{}
	actual := srv.GetServicesFromEnvVars()

	s.Len(*actual, 1)
	s.Contains(*actual, service)
}

// Mocks

type ReloadMock struct {
	ExecuteMock func(recreate bool) error
}

func MockReload(mock ReloadMock) func() {
	original := actions.NewReload
	actions.NewReload = func() actions.Reloader {
		return &mock
	}
	return func() { actions.NewReload = original }
}

func (m ReloadMock) Execute(recreate bool) error {
	return m.ExecuteMock(recreate)
}

type RemoveMock struct {
	mock.Mock
}

func (m *RemoveMock) Execute(args []string) error {
	params := m.Called(args)
	return params.Error(0)
}

func getRemoveMock(skipMethod string) *RemoveMock {
	mockObj := new(RemoveMock)
	if skipMethod != "Execute" {
		mockObj.On("Execute", mock.Anything).Return(nil)
	}
	return mockObj
}

type ReconfigureMock struct {
	ExecuteMock      func(reloadAfter bool) error
	GetDataMock      func() (actions.BaseReconfigure, proxy.Service)
	GetTemplatesMock func() (front, back string, err error)
}

func (m ReconfigureMock) Execute(reloadAfter bool) error {
	return m.ExecuteMock(reloadAfter)
}

func (m ReconfigureMock) GetData() (actions.BaseReconfigure, proxy.Service) {
	return m.GetDataMock()
}

func (m ReconfigureMock) GetTemplates() (front, back string, err error) {
	return m.GetTemplatesMock()
}

type FetchMock struct {
	ReloadServicesFromRegistryMock func(addresses []string, instanceName, mode string) error
	ReloadClusterConfigMock        func(listenerAddr string) error
	ReloadConfigMock               func(baseData actions.BaseReconfigure, mode string, listenerAddr string) error
}

func (m *FetchMock) ReloadServicesFromRegistry(addresses []string, instanceName, mode string) error {
	return m.ReloadServicesFromRegistryMock(addresses, instanceName, mode)
}
func (m FetchMock) ReloadClusterConfig(listenerAddr string) error {
	return m.ReloadClusterConfigMock(listenerAddr)
}
func (m FetchMock) ReloadConfig(baseData actions.BaseReconfigure, mode string, listenerAddr string) error {
	return m.ReloadConfigMock(baseData, mode, listenerAddr)
}
func MockFetch(mock FetchMock) func() {
	newFetchOrig := actions.NewFetch
	actions.NewFetch = func(baseData actions.BaseReconfigure, mode string) actions.Fetchable {
		return &mock
	}
	return func() {
		actions.NewFetch = newFetchOrig
	}
}

type CertMock struct {
	PutMock     func(http.ResponseWriter, *http.Request) (string, error)
	PutCertMock func(certName string, certContent []byte) (string, error)
	GetAllMock  func(w http.ResponseWriter, req *http.Request) (CertResponse, error)
	GetInitMock func() error
}

func (m CertMock) Put(w http.ResponseWriter, req *http.Request) (string, error) {
	return m.PutMock(w, req)
}

func (m CertMock) PutCert(certName string, certContent []byte) (string, error) {
	return m.PutCertMock(certName, certContent)
}

func (m CertMock) GetAll(w http.ResponseWriter, req *http.Request) (CertResponse, error) {
	return m.GetAllMock(w, req)
}

func (m CertMock) Init() error {
	return m.GetInitMock()
}
