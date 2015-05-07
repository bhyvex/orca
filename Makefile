SHA := $(shell git rev-parse HEAD)
OUTROOT := packaging

all: build

complete: build embed

build: gateway orcaman orcacli embed

gateway:
	go build -o $(OUTROOT)/sshgw -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/gateway

orcaman:
	go build -o $(OUTROOT)/orcaman -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/orcaman
	
orcacli:
	go build -o $(OUTROOT)/orcacli -ldflags "-X main.revision $(SHA)" github.com/clusterit/orca/cmd/cli

# to embed the resources we need bower and rice in the path
embed:
	cd cmd/orcaman && bower install
	rice --import-path="github.com/clusterit/orca/cmd/orcaman" append --exec="$(OUTROOT)/orcaman"

depends:
	go get github.com/GeertJohan/go.rice/rice
	go get github.com/robfig/glock
	glock sync github.com/clusterit/orca

glock:
	glock save github.com/clusterit/orca

testdrive:
	docker build -t orca .

runtestdrive:
	docker run -p 9011:9011 -p 2022:2022 -v $(HOME)/tmp/data:/data -e CLIENTID=$(GITHUB_CLIENTID) -e CLIENTSECRET=$(GITHUB_CLIENTSECRET) -e USERID=ulrichSchreiner orca
	
clean:
	rm -rf $(OUTROOT)/*
