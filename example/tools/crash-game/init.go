package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// RpcCfg rpc配置
type RpcCfg struct {
	TestNetUrl string `json:"test_net"`
	MainNetUrl string `json:"main_net"`
}

// JettonConfig jetton配置
type JettonConfig struct {
	MetaData         jetton.MetaData `json:"metadata"`
	JettonMinterAddr string          `json:"jetton_minter_addr"`
}

// CrashGameConfig crash game配置
type CrashGameConfig struct {
	ContractAddr    string `json:"crash_game_addr"`
	RoundNum        uint64 `json:"round_num"`
	MinIntervalTime uint64 `json:"min_interval_time"` // 最大间隔时间,单位秒
	Bet             struct {
		Amount   string `json:"amount"`
		Multiple uint64 `json:"multiple"`
	} `json:"bet"`
	Withdraw struct {
		Amount string `json:"amount"`
	} `json:"withdraw"`
}

type GlobalConfig struct {
	Rpc          RpcCfg          `json:"rpc"`
	Jetton       JettonConfig    `json:"jetton"`
	CrashGameCfg CrashGameConfig `json:"crash_game"`
}

// api client instance
var (
	TonAPI    ton.APIClientWrapped
	modelOnce sync.Once
)

// IsMainNet main net flag
var IsMainNet = flag.Bool("mainnet", false, "use main net")
var configPath = flag.String("config", "", "config file path")

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

func GetJettonMetaData() (error, *jetton.MetaData) {
	// 生成jetton minter合约的content
	cfg, err := GetGlobalCfg()
	if nil != err {
		return err, nil
	}
	return nil, &cfg.Jetton.MetaData
}

func GetScanCfg() string {
	if *IsMainNet {
		return MainNetScan
	} else {
		return TestNetScan
	}
}

func GetGlobalCfg() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	if *configPath == "" {
		dir, _ := os.Getwd()
		*configPath = fmt.Sprintf("%s/example/tools/crash-game/config.json", dir)
	}
	fmt.Printf("global config file path: %s\n", *configPath)
	data, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Println("read global config file failed:", err)
		return nil, err
	}
	if err = json.Unmarshal(data, cfg); nil != err {
		fmt.Println("unmarshal global config failed:", err)
		return nil, err
	}

	return cfg, nil
}

// UpdateGlobalCfg 更新全局配置，将配置写入config.json文件
func UpdateGlobalCfg(cfg *GlobalConfig) error {
	if *configPath == "" || cfg == nil {
		errMsg := "config path or config data is empty"
		fmt.Println(errMsg)
		return fmt.Errorf(errMsg)
	}
	updatedJSON, err := json.MarshalIndent(*cfg, "", "    ")
	if err != nil {
		fmt.Println("marshal global config failed:", err)
		return err
	}
	// 写回文件
	if err := ioutil.WriteFile(*configPath, updatedJSON, 0644); err != nil {
		fmt.Println("Error writing jetton config to file:", err)
		return err
	}

	fmt.Println("JSON file updated successfully.")
	return nil
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
