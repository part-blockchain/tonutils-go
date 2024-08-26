package main

import (
	"context"
	"flag"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"log"
	"sync"
)

// api client instance
var (
	TonAPI    ton.APIClientWrapped
	modelOnce sync.Once
)

// IsMainNet main net flag
var IsMainNet = flag.Bool("mainnet", false, "use main net")

// 初始化api client单例对象
func initTonApiIns() ton.APIClientWrapped {
	flag.Parse()
	client := liteclient.NewConnectionPool()
	netCfgUrl := TestNetCfg
	if *IsMainNet {
		netCfgUrl = MainNetCfg
		log.Println("this is Main Net Config")
	} else {
		log.Println("this is Test Net Config")
	}
	// get config
	cfg, err := liteclient.GetConfigFromUrl(context.Background(), netCfgUrl)
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
		return nil
	}

	// connect to mainnet lite servers
	err = client.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return nil
	}

	// api client with full proof checks
	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(cfg)
	return api
}

// GetTonAPIIns Getting the global ton api and context single instance
func GetTonAPIIns() ton.APIClientWrapped {
	modelOnce.Do(func() {
		TonAPI = initTonApiIns()
	})
	return TonAPI
}

func GetRpcUrl() string {
	if *IsMainNet {
		return MainNetUrl
	} else {
		return TestNetUrl
	}
}

func GetApiKey() string {
	if *IsMainNet {
		return MainNetApiKey
	} else {
		return TestNetApiKey
	}
}
