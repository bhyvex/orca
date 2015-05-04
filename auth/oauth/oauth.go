package oauth

import (
	"fmt"

	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/rest"
	"github.com/clusterit/orca/users"
	"gopkg.in/emicklei/go-restful.v1"
)

const (
	oauthPath              = "/oauth"
	typeOauth ProviderType = "oauth"
	typeBasic              = "basic"
)

type ProviderType string

type AuthRegistration struct {
	Type           ProviderType `json:"type"`
	Network        string       `json:"network"`
	ClientId       string       `json:"clientid"`
	ClientSecret   string       `json:"clientsecret"`
	Scopes         string       `json:"scopes"`
	AuthUrl        string       `json:"auth_url"`
	AccessTokenUrl string       `json:"accesstoken_url"`
	UserinfoUrl    string       `json:"userinfo_url"`
	PathId         string       `json:"pathid"`
	PathName       string       `json:"pathname"`
	PathPicture    string       `json:"pathpicture"`
	PathCover      string       `json:"pathcover"`
}

type LoginProvider struct {
	Type     ProviderType `json:"type"`
	Network  string       `json:"network"`
	ClientId string       `json:"clientid"`
	Scopes   string       `json:"scopes"`
	AuthUrl  string       `json:"authurl"`
}

type AuthRegistry interface {
	Create(tp ProviderType, network string, clientid, clientsecrect, scopes, authurl, accessurl, userinfourl, pathid, pathname, pathpicture, pathcover string) (*AuthRegistration, error)
	Delete(network string) (*AuthRegistration, error)
	Get(network string) (*AuthRegistration, error)
	GetAll() ([]AuthRegistration, error)
}

type AuthRegService struct {
	Auth     auth.Auther
	Users    users.Users
	Registry AuthRegistry
}

type oauthApp struct {
	cc      *etcd.Cluster
	persist etcd.Persister
}

func New(cc *etcd.Cluster) (AuthRegistry, error) {
	pers, err := cc.NewJsonPersister(oauthPath)
	if err != nil {
		return nil, err
	}
	return &oauthApp{cc, pers}, nil
}

func (a *oauthApp) Get(network string) (*AuthRegistration, error) {
	var res AuthRegistration
	return &res, a.persist.Get(network, &res)
}

func (a *oauthApp) Create(tp ProviderType, network, clientid, clientsecret, scopes, authurl, accessurl, userinfourl, pathid, pathname, pathpicture, pathcover string) (*AuthRegistration, error) {
	if network == "" {
		return nil, fmt.Errorf("empty network not allowed")
	}
	reg := AuthRegistration{
		Type:           tp,
		Network:        network,
		ClientId:       clientid,
		ClientSecret:   clientsecret,
		Scopes:         scopes,
		AuthUrl:        authurl,
		AccessTokenUrl: accessurl,
		UserinfoUrl:    userinfourl,
		PathId:         pathid,
		PathName:       pathname,
		PathPicture:    pathpicture,
		PathCover:      pathcover,
	}
	// if this is a known network and there are empty fields, fill them ...
	reg = fillDefaults(network, reg)
	a.persist.Put(network, reg)
	return &reg, nil
}

func (a *oauthApp) Delete(network string) (*AuthRegistration, error) {
	var res AuthRegistration
	if err := a.persist.Get(network, &res); err != nil {
		return nil, err
	}
	return &res, a.persist.Remove(network)
}

func (a *oauthApp) GetAll() ([]AuthRegistration, error) {
	var res []AuthRegistration
	return res, a.persist.GetAll(true, false, &res)
}

func (t *AuthRegService) Shutdown() {
}

func (t *AuthRegService) Register(root string, c *restful.Container) {

	mgr := users.CheckUser(t.Auth, t.Users, users.ManagerRoles, nil)

	ws := new(restful.WebService)
	ws.
		Path(root + "authregistry").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("/").To(mgr(t.createReg)).
		Doc("create a oauth registration").
		Operation("createReg").
		Reads(AuthRegistration{}).
		Writes(AuthRegistration{}))
	ws.Route(ws.GET("/").To(mgr(t.getAllRegs)).
		Doc("get all registered oauth registrations").
		Operation("getAllRegs").
		Returns(200, "OK", []AuthRegistration{}))
	ws.Route(ws.GET("/loginProviders").To(t.loginProviders).
		Doc("get all login providers usable for login").
		Operation("loginProviders").
		Returns(200, "OK", []LoginProvider{}))
	ws.Route(ws.DELETE("/{network}").To(mgr(t.deleteReg)).
		Doc("delete the registry for the given network").
		Param(ws.PathParameter("network", "the network name of the registry").DataType("string")).
		Operation("deleteReg").
		Returns(200, "OK", AuthRegistration{}))

	c.Add(ws)
}

func (t *AuthRegService) createReg(me *users.User, request *restful.Request, response *restful.Response) {
	var reg AuthRegistration
	if err := request.ReadEntity(&reg); err != nil {
		rest.HandleError(err, response)
		return
	}
	rest.HandleEntity(t.Registry.Create(
		ProviderType(reg.Type),
		reg.Network,
		reg.ClientId,
		reg.ClientSecret,
		reg.Scopes,
		reg.AuthUrl,
		reg.AccessTokenUrl,
		reg.UserinfoUrl,
		reg.PathId,
		reg.PathName,
		reg.PathPicture,
		reg.PathCover))(request, response)
}

func (t *AuthRegService) deleteReg(me *users.User, request *restful.Request, response *restful.Response) {
	network := request.PathParameter("network")
	rest.HandleEntity(t.Registry.Delete(network))(request, response)
}

func (t *AuthRegService) getAllRegs(me *users.User, request *restful.Request, response *restful.Response) {
	rest.HandleEntity(t.Registry.GetAll())(request, response)
}

func (t *AuthRegService) loginProviders(request *restful.Request, response *restful.Response) {
	regs, err := t.Registry.GetAll()
	if err != nil {
		rest.HandleError(err, response)
		return
	}
	provs := make([]LoginProvider, len(regs))
	for i, r := range regs {
		provs[i].Network = r.Network
		provs[i].ClientId = r.ClientId
		provs[i].Scopes = r.Scopes
		provs[i].AuthUrl = r.AuthUrl
	}
	response.WriteEntity(provs)
}
