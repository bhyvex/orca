package service

import (
	"github.com/clusterit/orca/auth"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/rest"
	"github.com/clusterit/orca/users"
	"gopkg.in/emicklei/go-restful.v1"
)

type ConfigService struct {
	Zone   string
	Auth   auth.Auther
	Users  users.Users
	Config config.Configer
}

func (t *ConfigService) Shutdown() error {
	return nil
}

func (t *ConfigService) Register(root string, c *restful.Container) {
	ws := new(restful.WebService)

	mgr := users.CheckUser(t.Auth, t.Users, users.ManagerRoles, nil)

	ws.
		Path(root + "configuration").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/cluster").To(mgr(t.getClusterConfig)).
		Doc("Get the cluster config").
		Operation("getClusterConfig").
		Writes(config.ClusterConfig{}))
	ws.Route(ws.PUT("/cluster").To(mgr(t.setClusterConfig)).
		Doc("Set the cluster config").
		Operation("setClusterConfig").
		Reads(config.ClusterConfig{}).
		Writes(config.ClusterConfig{}))
	ws.Route(ws.PUT("/{zone}/gateway").To(mgr(t.putGateway)).
		Doc("Store the Gateway config for a given zone").
		Param(ws.PathParameter("zone", "the stzoneage to put the Gateway config to").DataType("string")).
		Operation("putGateway").
		Reads(config.Gateway{}))
	ws.Route(ws.GET("/{zone}/gateway").To(mgr(t.getGateway)).
		Doc("Get the Gateway config for a given zone").
		Param(ws.PathParameter("zone", "the zone to read the JWT from").DataType("string")).
		Operation("getGateway").
		Writes(config.Gateway{}))
	ws.Route(ws.GET("/zones").To(mgr(t.getZones)).
		Doc("Get all current configured zones").
		Operation("getZones").
		Writes([]string{}))
	ws.Route(ws.GET("/zone").To(t.getZone).
		Doc("Get the current zone").
		Operation("getZone").
		Writes(""))
	ws.Route(ws.PUT("/zone/{zone}").To(mgr(t.putZone)).
		Doc("Create a new zone").
		Param(ws.PathParameter("zone", "the zone to create").DataType("string")).
		Operation("putZone").
		Writes(""))
	ws.Route(ws.DELETE("/zone/{zone}").To(mgr(t.deleteZone)).
		Doc("Drop a zone").
		Param(ws.PathParameter("zone", "the zone to create").DataType("string")).
		Operation("deleteZone").
		Writes(""))

	c.Add(ws)
}

func (t *ConfigService) getZones(u *users.User, rq *restful.Request, rsp *restful.Response) {
	rest.HandleEntity(t.Config.Zones())(rq, rsp)
}

func (t *ConfigService) getZone(rq *restful.Request, rsp *restful.Response) {
	rest.HandleEntity(t.Zone, nil)(rq, rsp)
}

func (t *ConfigService) putZone(u *users.User, rq *restful.Request, rsp *restful.Response) {
	z := rq.PathParameter("zone")
	if err := t.Config.CreateZone(z); err != nil {
		rest.HandleError(err, rsp)
		return
	}
	_, err := config.InitZone(t.Config, z, true)
	if err != nil {
		t.Config.DropZone(z)
		rest.HandleError(err, rsp)
		return
	}
	rsp.WriteEntity(z)
}

func (t *ConfigService) deleteZone(u *users.User, rq *restful.Request, rsp *restful.Response) {
	z := rq.PathParameter("zone")
	if err := t.Config.DropZone(z); err != nil {
		rest.HandleError(err, rsp)
		return
	}
	rsp.WriteEntity(z)
}

func (t *ConfigService) putGateway(u *users.User, rq *restful.Request, rsp *restful.Response) {
	var gw config.Gateway
	z := rq.PathParameter("zone")
	err := rq.ReadEntity(&gw)
	if err != nil {
		rest.HandleError(err, rsp)
		return
	}
	if err := t.Config.PutGateway(z, gw); err != nil {
		rest.HandleError(err, rsp)
		return
	}
	rsp.WriteEntity(gw)
}

func (t *ConfigService) getGateway(u *users.User, rq *restful.Request, rsp *restful.Response) {
	z := rq.PathParameter("zone")
	rest.HandleEntity(t.Config.GetGateway(z))(rq, rsp)
}

func (t *ConfigService) getClusterConfig(u *users.User, rq *restful.Request, rsp *restful.Response) {
	rest.HandleEntity(t.Config.Cluster())(rq, rsp)
}

func (t *ConfigService) setClusterConfig(u *users.User, rq *restful.Request, rsp *restful.Response) {
	var cf config.ClusterConfig
	err := rq.ReadEntity(&cf)
	if err != nil {
		rest.HandleError(err, rsp)
		return
	}
	rest.HandleEntity(t.Config.UpdateCluster(cf))(rq, rsp)
}
