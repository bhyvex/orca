package main

import (
	"fmt"
	"io/ioutil"

	"github.com/clusterit/orca/auth/oauth"
	"github.com/clusterit/orca/config"
	"github.com/clusterit/orca/users"

	"github.com/jmcvetta/napping"
)

func (c *cli) unmarshal(rq *napping.Request, target interface{}) error {
	resp, err := c.session.Send(rq)
	if err != nil {
		return err
	}
	if resp.Status() != 200 {
		return fmt.Errorf("HTTP %d: %s", resp.Status(), resp.RawText())
	}
	if target != nil {
		return resp.Unmarshal(target)
	}
	return nil
}

func (c *cli) createUser(network, id, name string, roles ...string) error {
	rlz := make([]users.Role, len(roles))
	for i, r := range roles {
		rlz[i] = users.Role(r)
	}
	t := users.User{Id: id, Name: name, Roles: rlz}
	r := c.rq("PUT", "/api/users/"+network, t)
	return c.unmarshal(r, nil)
}

func (c *cli) listUsers() ([]users.User, error) {
	var res []users.User
	r := c.rq("GET", "/api/users", nil)
	return res, c.unmarshal(r, &res)
}

func (c *cli) me() (*users.User, error) {
	var res users.User
	r := c.rq("GET", "/api/users/me", nil)
	return &res, c.unmarshal(r, &res)
}

func (c *cli) permit(dur int) (*users.Allowance, error) {
	var res users.Allowance
	r := c.rq("PATCH", fmt.Sprintf("/api/users/permit/%d", dur), nil)
	return &res, c.unmarshal(r, &res)
}

func (c *cli) parseKey(k string) (*users.Key, error) {
	var key users.Key
	r := c.rq("POST", "/api/users/parsekey", k)
	return &key, c.unmarshal(r, &key)
}

func (c *cli) addKey(uid, keyname string, file string) error {
	kf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if keyname == "" {
		k, err := c.parseKey(string(kf))
		if err != nil {
			return err
		}
		keyname = k.Id
	}
	// zone hardcoded, cause orca don't uses zones for users
	r := c.rq("PUT", fmt.Sprintf("/api/users/%s/%s/zone/pubkey", uid, keyname), string(kf))
	return c.unmarshal(r, nil)
}

func (c *cli) deleteKey(uid, keyname string) error {
	// zone hardcoded, cause orca don't uses zones for users
	r := c.rq("DELETE", fmt.Sprintf("/api/users/%s/%s/zone/pubkey", uid, keyname), nil)
	return c.unmarshal(r, nil)
}

func (c *cli) zones() ([]string, error) {
	var res []string
	r := c.rq("GET", "/api/configuration/zones", nil)
	return res, c.unmarshal(r, &res)
}

func (c *cli) getGateway(stage string) (*config.Gateway, error) {
	var res config.Gateway
	r := c.rq("GET", fmt.Sprintf("/api/configuration/%s/gateway", stage), nil)
	return &res, c.unmarshal(r, &res)
}

func (c *cli) putGateway(stage string, gw config.Gateway) error {
	r := c.rq("PUT", fmt.Sprintf("/api/configuration/%s/gateway", stage), gw)
	return c.unmarshal(r, nil)
}

func (c *cli) getCluster() (*config.ClusterConfig, error) {
	var res config.ClusterConfig
	r := c.rq("GET", fmt.Sprintf("/api/configuration/cluster"), nil)
	return &res, c.unmarshal(r, &res)
}

func (c *cli) putCluster(cc config.ClusterConfig) error {
	r := c.rq("PUT", fmt.Sprintf("/api/configuration/cluster"), cc)
	return c.unmarshal(r, nil)
}
func (c *cli) listOauthProviders() ([]oauth.AuthRegistration, error) {
	var res []oauth.AuthRegistration
	r := c.rq("GET", "/api/authregistry", nil)
	return res, c.unmarshal(r, &res)
}

func (c *cli) putProvider(p oauth.AuthRegistration) error {
	r := c.rq("PUT", "/api/authregistry", p)
	return c.unmarshal(r, nil)
}
func (c *cli) delProvider(n string) error {
	r := c.rq("DELETE", "/api/authregistry/"+n, nil)
	return c.unmarshal(r, nil)
}

func (c *cli) addAlias(network, alias string) error {
	r := c.rq("PUT", "/api/users/alias/"+network+"/"+alias, nil)
	return c.unmarshal(r, nil)
}
func (c *cli) removeAlias(network, alias string) error {
	r := c.rq("DELETE", "/api/users/alias/"+network+"/"+alias, nil)
	return c.unmarshal(r, nil)
}
