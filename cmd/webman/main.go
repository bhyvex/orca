package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strings"

	"github.com/clusterit/orca/logging"

	"github.com/clusterit/orca/cmd"
	"github.com/clusterit/orca/config"

	"github.com/clusterit/orca/auth"
	_ "github.com/clusterit/orca/auth/google"
	"github.com/clusterit/orca/auth/jwt"
	"github.com/clusterit/orca/etcd"
	"github.com/clusterit/orca/users"
	uetcd "github.com/clusterit/orca/users/etcd"

	"github.com/GeertJohan/go.rice"
	"github.com/spf13/cobra"
	"gopkg.in/emicklei/go-restful.v1"
)

var (
	etcdConfig string
	listen     string
	publish    string
	zone       string
	logger     = logging.Simple()
	revision   string
)

type webmanager struct {
	cluster       *etcd.Cluster
	userimpl      users.Users
	authimpl      auth.Auther
	configimpl    config.Configer
	autherService *auth.AutherService
	usersService  *users.UsersService
	configService *config.ConfigService
}

var webman = &cobra.Command{Use: "webman"}

func NewWebManager(etcds []string) (*webmanager, error) {
	cc, err := etcd.Init(etcds)
	if err != nil {
		return nil, err
	}
	userimpl, err := uetcd.New(cc)
	if err != nil {
		return nil, err
	}

	cfger, err := config.New(cc)
	if err != nil {
		return nil, err
	}
	wm := &webmanager{cluster: cc,
		configimpl: cfger,
		userimpl:   userimpl,
	}
	if err := wm.initWithZone(zone); err != nil {
		return nil, err
	}
	return wm, nil
}
func (wm *webmanager) Stop() {
	wm.autherService.Shutdown()
	wm.usersService.Shutdown()
	wm.configService.Shutdown()
}

func (wm *webmanager) auth() *restful.Container {
	c := restful.NewContainer()
	wm.autherService = &auth.AutherService{Auth: wm.authimpl}
	wm.autherService.Register(c)
	return c
}

func (wm *webmanager) users() *restful.Container {
	c := restful.NewContainer()
	wm.usersService = &users.UsersService{Auth: wm.authimpl, Provider: wm.userimpl}
	wm.usersService.Register(c)

	return c
}

func (wm *webmanager) config() *restful.Container {
	c := restful.NewContainer()
	wm.configService = &config.ConfigService{Auth: wm.authimpl, Users: wm.userimpl, Config: wm.configimpl, Zone: zone}
	wm.configService.Register(c)

	return c
}

func (wm *webmanager) initWithZone(zone string) error {
	_, cfg, err := cmd.ForceZone(wm.configimpl, zone, true, true)
	if err != nil {
		return err
	}
	blk, _ := pem.Decode([]byte(cfg.Key))
	jwtPk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		return err
	}

	wm.authimpl = jwt.NewAuther(jwtPk)

	go func() {
		mgr, stp, err := wm.configimpl.ManagerConfig(zone)
		if err != nil {
			logger.Errorf("cannot create watcher for manger config config: %s", err)
			return
		}
		for m := range mgr {
			logger.Debugf("new manager config: Key:%s", m.Key)
			wm.switchSettings(m)
		}
		close(stp)
	}()
	return nil
}

func (cm *webmanager) switchSettings(cfg config.ManagerConfig) error {
	blk, _ := pem.Decode([]byte(cfg.Key))
	jwtPk, err := x509.ParsePKCS1PrivateKey(blk.Bytes)
	if err != nil {
		return err
	}

	cm.authimpl = jwt.NewAuther(jwtPk)
	cm.usersService.Auth = cm.authimpl
	cm.configService.Auth = cm.authimpl
	return nil
}

func (wm *webmanager) ServeAndPublish() {
	man, e := wm.cluster.NewManager()
	if e != nil {
		panic(e)
	}
	if publish != "" {
		man.Register("/"+cmd.ManagerService, publish, 20)
	}
	logger.Infof("start listening on %s", listen)
	http.ListenAndServe(listen, nil)
}
func main() {
	var cmdServe = &cobra.Command{
		Use:   "serve",
		Short: "Starts the web manager to listen on the given address",
		Long:  "Start the web manager service on the given address.",
		Run: func(cm *cobra.Command, args []string) {
			publish = cmd.PublishAddress(publish, listen)
			wm, err := NewWebManager(strings.Split(etcdConfig, ","))
			if err != nil {
				panic(err)
			}
			http.Handle("/auth/", wm.auth())
			http.Handle("/users/", wm.users())
			http.Handle("/configuration/", wm.config())
			http.Handle("/", http.FileServer(rice.MustFindBox("app").HTTPBox()))
			wm.ServeAndPublish()
			defer wm.Stop()
		},
	}

	webman.PersistentFlags().StringVarP(&etcdConfig, "etcd", "e", "http://localhost:4001", "etcd cluster machine Url's")
	webman.PersistentFlags().StringVarP(&publish, "publish", "p", "self", "self published http address. if empty don't publish, the value 'self' will be replace with the currnent listen address")
	webman.PersistentFlags().StringVarP(&zone, "zone", "z", "intranet", "use this zone as a subtree in the etcd backbone")
	cmdServe.Flags().StringVarP(&listen, "listen", "l", ":9011", "listen address for web manager")
	webman.AddCommand(cmdServe)
	webman.Execute()
}
