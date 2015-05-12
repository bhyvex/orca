package etcd

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/clusterit/orca/logging"

	"github.com/clusterit/orca/common"
	"github.com/clusterit/orca/etcd"
	. "github.com/clusterit/orca/users"
	etcderr "github.com/coreos/etcd/error"
	goetcd "github.com/coreos/go-etcd/etcd"
	"github.com/dgryski/dgoogauth"
)

const (
	usersPath  = "/users"
	aliasPath  = "/alias"
	keysPath   = "/keys"
	permitPath = "/permit"
	twofaPath  = "/2fa"
	idtoksPath = "/idtoks"
)

var (
	logger = logging.Simple()
)

type etcdUsers struct {
	up     etcd.Persister
	kp     etcd.Persister
	pm     etcd.Persister
	al     etcd.Persister
	twofa  etcd.Persister
	idtoks etcd.Persister

	// used for testing of 2FA
	scratchCodes []int
}

func New(cl *etcd.Cluster) (Users, error) {
	up, e := cl.NewJsonPersister("/data" + usersPath)
	if e != nil {
		return nil, e
	}
	kp, e := cl.NewJsonPersister("/data" + keysPath)
	if e != nil {
		return nil, e
	}
	pm, e := cl.NewJsonPersister("/data" + permitPath)
	if e != nil {
		return nil, e
	}
	al, e := cl.NewJsonPersister("/data" + aliasPath)
	if e != nil {
		return nil, e
	}
	twofa, e := cl.NewJsonPersister("/data" + twofaPath)
	if e != nil {
		return nil, e
	}
	idtoks, e := cl.NewJsonPersister("/data" + idtoksPath)
	if e != nil {
		return nil, e
	}
	return &etcdUsers{up: up, kp: kp, pm: pm, al: al, twofa: twofa, idtoks: idtoks}, nil
}

func (eu *etcdUsers) key(k *Key) string {
	return strings.Replace(k.Fingerprint, ":", "", -1)
}

func uid(net, id string) string {
	return common.NetworkUser(net, id)
}

func (eu *etcdUsers) RemoveAlias(id, network, alias string) (*User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, e
	}
	auid := uid(network, alias)
	u.Aliases = remove(auid, u.Aliases)
	if err := eu.al.Remove(auid); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) AddAlias(id, network, alias string) (*User, error) {
	u, e := eu.Get(id)
	if e != nil {
		return nil, e
	}
	auid := uid(network, alias)
	u.Aliases = insert(auid, u.Aliases)
	if err := eu.al.Put(auid, u.Id); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) Create(network, id, name string, rlz Roles) (*User, error) {
	usrid := id
	if network != "" {
		usrid = uid(network, id)
	}
	u, e := eu.Get(usrid)
	if e != nil {
		internalid := common.GenerateUUID()
		idtoken := common.GenerateUUID()
		u = &User{Id: internalid, Name: name, Roles: rlz, Aliases: []string{usrid}, IdToken: idtoken}
		// generate an alias for internalid too
		if err := eu.al.Put(internalid, internalid); err != nil {
			return nil, err
		}
	} else {
		u.Name = name
		u.Roles = rlz
		u.Allowance = nil
		u.Aliases = insert(usrid, u.Aliases)
	}
	if err := eu.al.Put(usrid, u.Id); err != nil {
		return nil, err
	}
	if err := eu.idtoks.Put(u.IdToken, u.Id); err != nil {
		return nil, err
	}
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) NewIdToken(uid string) (*User, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return nil, e
	}
	// ignore error when removing old token
	idtoken := common.GenerateUUID()
	if err := eu.idtoks.Put(idtoken, u.Id); err != nil {
		return nil, err
	}
	eu.idtoks.Remove(u.IdToken)
	u.IdToken = idtoken
	return u, eu.up.Put(u.Id, &u)
}

func (eu *etcdUsers) ByIdToken(idtok string) (*User, error) {
	var uid string
	if err := eu.idtoks.Get(idtok, &uid); err != nil {
		return nil, wrapError(err)
	}
	return eu.Get(uid)
}

func (eu *etcdUsers) GetAll() ([]User, error) {
	var res []User
	return res, eu.up.GetAll(true, false, &res)
}

func (eu *etcdUsers) Get(id string) (*User, error) {
	var u User
	var a Allowance
	var realid string
	// we have an alias for our intenal id too, so the
	// next lookup must always succeed if the user exists
	if err := eu.al.Get(id, &realid); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, common.ErrNotFound
			}
		}
		return nil, err
	}
	if err := eu.up.Get(realid, &u); err != nil {
		if cerr, ok := err.(*goetcd.EtcdError); ok {
			if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
				return nil, common.ErrNotFound
			}
		}
		return nil, err
	}
	if err := eu.pm.Get(realid, &a); err == nil {
		u.Allowance = &a
	} else {
		u.Allowance = nil
	}
	return &u, nil
}

func (eu *etcdUsers) GetByKey(pubkey string) (*User, *Key, error) {
	var (
		u   User
		uid string
	)
	pk, err := ParseKey(pubkey)
	if err != nil {
		return nil, nil, err
	}
	if err := eu.kp.Get(eu.key(pk), &uid); err != nil {
		return nil, nil, wrapError(err)
	}
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, nil, wrapError(err)
	}
	for _, k := range u.Keys {
		if pk.Value == k.Value {
			var a Allowance
			if err := eu.pm.Get(u.Id, &a); err == nil {
				u.Allowance = &a
			} else {
				u.Allowance = nil
			}
			return &u, &k, nil
		}
	}
	return nil, nil, common.ErrNotFound
}

func (eu *etcdUsers) AddKey(uid, kid string, pubkey string, fp string) (*Key, error) {
	k := Key{Id: kid, Fingerprint: fp, Value: pubkey}
	u, err := eu.Get(uid)
	if err != nil {
		return nil, err
	}
	uid = u.Id
	var found *Key
	for _, k := range u.Keys {
		if k.Fingerprint == fp {
			found = &k
			break
		}
	}
	if found != nil {
		// key with same FP already exists, do nothing
		return found, nil
	}
	u.Keys = append(u.Keys, k)
	if err := eu.up.Put(uid, &u); err != nil {
		return nil, err
	}
	if err := eu.kp.Put(eu.key(&k), uid); err != nil {
		// we should put the old user back ... i'm too lazy now
		return nil, err
	}
	return &k, nil
}

func (eu *etcdUsers) RemoveKey(uid, kid string) (*Key, error) {
	u, err := eu.Get(uid)
	if err != nil {
		return nil, err
	}
	uid = u.Id
	var newkeys []Key
	var found Key

	for i, k := range u.Keys {
		if k.Id != kid {
			newkeys = append(newkeys, u.Keys[i])
		} else {
			found = k
		}
	}
	u.Keys = newkeys
	if err := eu.kp.Remove(eu.key(&found)); err != nil {
		return nil, err
	}
	return &found, eu.up.Put(uid, &u)
}

func (eu *etcdUsers) Update(uid, username string, rolz Roles) (*User, error) {
	u, err := eu.Get(uid)
	if err != nil {
		return nil, err
	}
	u.Name = username
	u.Roles = rolz
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) Permit(a Allowance, ttlSecs uint64) error {
	u, err := eu.Get(a.Uid)
	if err != nil {
		return err
	}
	uid := u.Id
	if ttlSecs == 0 {
		logger.Infof("remove allowance for %s", uid)
		return eu.pm.Remove(uid)
	}
	a.Until = time.Now().UTC().Add(time.Second * time.Duration(ttlSecs))
	return eu.pm.PutTtl(uid, ttlSecs, &a)
}

func (eu *etcdUsers) Delete(uid string) (*User, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return nil, e
	}
	uid = u.Id
	if err := eu.up.Get(uid, &u); err != nil {
		return nil, err
	}
	for _, k := range u.Keys {
		if err := eu.kp.Remove(eu.key(&k)); err != nil {
			return nil, err
		}
	}
	eu.pm.Remove(uid)
	return u, eu.up.Remove(uid)
}

func (eu *etcdUsers) Create2FAToken(domain, uid string) (string, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return "", e
	}
	sec := make([]byte, 6)
	_, err := rand.Read(sec)
	if err != nil {
		return "", err
	}
	encodedSecret := base32.StdEncoding.EncodeToString(sec)
	if err := eu.twofa.Put(uid, encodedSecret); err != nil {
		return "", err
	}
	auth_string := "otpauth://totp/" + u.Name + "@" + domain + "?secret=" + encodedSecret + "&issuer=orca"
	return auth_string, nil
}

func (eu *etcdUsers) CheckAndAllowToken(uid, token string, maxAllowance int) error {
	if err := eu.CheckToken(uid, token); err != nil {
		return err
	}
	u, e := eu.Get(uid)
	if e != nil {
		return e
	}
	uid = u.Id
	permit := u.AutologinAfter2FA
	if maxAllowance < permit {
		permit = maxAllowance
	}
	if permit > 0 {
		a := Allowance{
			GrantedBy: uid,
			Uid:       uid,
			Until:     time.Now(), // will be set in the Permit function
		}
		return eu.Permit(a, uint64(permit))
	}
	return nil
}

func (eu *etcdUsers) CheckToken(uid, token string) error {
	var secret string
	if err := eu.twofa.Get(uid, &secret); err != nil {
		return err
	}

	otpc := &dgoogauth.OTPConfig{
		Secret:       secret,
		WindowSize:   3,
		HotpCounter:  0,
		ScratchCodes: eu.scratchCodes,
	}
	ok, err := otpc.Authenticate(token)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid token")
	}
	return nil
}

func (eu *etcdUsers) Use2FAToken(uid string, use bool) error {
	u, e := eu.Get(uid)
	if e != nil {
		return e
	}
	u.Use2FA = use
	if !use {
		eu.twofa.Remove(u.Id)
	}
	return eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) SetAutologinAfter2FA(uid string, duration int) (*User, error) {
	u, e := eu.Get(uid)
	if e != nil {
		return nil, e
	}
	u.AutologinAfter2FA = duration
	return u, eu.up.Put(u.Id, u)
}

func (eu *etcdUsers) Close() error {
	return nil
}

func insert(s string, ar []string) []string {
	m := make(map[string]bool)
	for _, a := range ar {
		m[a] = true
	}
	m[s] = true
	var res []string
	for k, _ := range m {
		res = append(res, k)
	}
	return res
}

func remove(s string, ar []string) []string {
	m := make(map[string]bool)
	for _, a := range ar {
		m[a] = true
	}
	delete(m, s)
	var res []string
	for k, _ := range m {
		res = append(res, k)
	}
	return res
}

func wrapError(e error) error {
	if cerr, ok := e.(*goetcd.EtcdError); ok {
		if cerr.ErrorCode == etcderr.EcodeKeyNotFound {
			return common.ErrNotFound
		}
	}
	return e
}
