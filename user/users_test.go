package user

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	testpkvalue   = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDyLg8zuWzJOgcTru78NkhhDsa+tjasjrJGoJbhBRMHrxgwdgUF5ZKGsV2LWTZgp8rIDUjHRGWSlvTrpXCG33wmRJrXwxYG3J0QeOAYRlMD3ESBVtPWm2iqA02PzpL7+mnmV79Ml3Q8yUz8Ef5Bs+lytVAw42IhfTEfJyWM9zsjFEW/NvZ6cttrOUhwEQ1r9HvY0UDyHRA3sW0B3I2KfYg1Z1e5wlKDd7dGI9u/S9E9JwFpeh/AXjPiN/Vd2xInIh99G9HsWBdpTaNlYXZj6Qnx/wLcCm2v7U9WdIvM5M+xqiYZ6pxGUtsBDgBjraxh8tRWV3eab3stZsKnwQthyp4P"
	testpkpubkey  = testpkvalue + " title"
	testpkfp      = "f2:97:4e:2f:9e:a8:52:cd:c1:6d:62:f3:a7:69:b5:cc"
	testpk2value  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDyLg8zuWzJOgcTru78NkhhDsa+tjasjrJGoJbhBRMHrxgwdgUF5ZKGsV2LWTZgp8rIDUjHRGWSlvTrpXCG33wmRJrXwxYG3J0QeOAYRlMD3ESBVtPWm2iqA02PzpL7+mnmV79Ml3Q8yUz8Ef5Bs+lytVAw42IhfTEfJyWM9zsjFEW/NvZ6cttrOUhwEQ1r9HvY0UDyHRA3sW0B3I2KfYg1Z1e5wlKDd7dGI9u/S9E9JwFpeh/AXjPiN/Vd2xInIh99G9HsWBdpTaNlYXZj6Qnx/wLcCm2v7U9WdIvM5M+xqiYZ6pxGUtsBDgBjraxh8tRWV3eab3stZsKnwQthyp4P"
	testpk2pubkey = testpk2value + " title2"
	testpk2fp     = "f2:97:4e:2f:9e:a8:52:cd:c1:6d:62:f3:a7:69:b5:cc"
)

func TestSSHKeys(t *testing.T) {
	Convey("Parseing ssh-keys", t, func() {
		k, e := ParseKey(testpkpubkey)
		So(e, ShouldBeNil)
		Convey("the title should be ok", func() {
			So(k.ID, ShouldEqual, "title")
		})
		Convey("as the fingerprint", func() {
			So(k.Fingerprint, ShouldEqual, testpkfp)

		})
		Convey("and the value itself", func() {
			So(k.Value, ShouldEqual, testpkvalue)
		})
	})
}

func TestRoles(t *testing.T) {
	Convey("create role arrays", t, func() {
		r := Roles{Role("a"), Role("b"), Role("c")}
		So(r.String(), ShouldEqual, "a,b,c")
		Convey("check if 'a' is in roles", func() {
			So(r.Has(Role("a")), ShouldBeTrue)
		})
		Convey("and 'd' is not", func() {
			So(r.Has(Role("d")), ShouldBeFalse)
		})
	})
}
