package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	CrashGame "github.com/xssnick/tonutils-go/ton/crash-game"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
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
	ContractAddr      string `json:"crash_game_addr"`
	MaxRoundsParallel uint64 `json:"max_rounds_parallel"` // 最大并行游戏轮数
	RoundNum          uint64 `json:"round_num"`
	MinIntervalTime   uint64 `json:"min_interval_time"` // 最大间隔时间,单位秒
	Bet               struct {
		Amount   string `json:"amount"`
		Multiple uint64 `json:"multiple"`
	} `json:"bet"`
	Withdraw struct {
		Amount string `json:"amount"`
	} `json:"withdraw"`
}

// ParamsConfig 参数配置
type ParamsConfig struct {
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

// CallEnvCfg 合约Call调用的环境配置
type CallEnvCfg struct {
	// CrashGame合约地址
	CrashGameAddr *address.Address
	// 参数配置
	ParamsCfg *ParamsConfig
	// 钱包对象
	Wallet *wallet.Wallet
	// Jetton客户端对象
	JettonClient *jetton.Client
	// CrashGame客户端对象
	CrashGameClient *CrashGame.Client
	// GameWallet客户端对象
	GameWalletClient *CrashGame.GameWalletClient
}

// 环境准备
func prepareEnvCfg(walletIndex int) *CallEnvCfg {
	callEvnCfg := &CallEnvCfg{}
	// 基础环境
	cfg, w := prepareBaseEnv(walletIndex)
	callEvnCfg.ParamsCfg = cfg
	callEvnCfg.Wallet = w

	// 业务环境
	// 获取Jetton客户端

	return callEvnCfg
}

// 环境准备
func prepareBaseEnv(walletIndex int) (*ParamsConfig, *wallet.Wallet) {
	// read from config file
	cfg, err := GetParamsCfg()
	if nil != err {
		log.Fatal("get global config failed")
	}

	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			log.Fatal("get ton api instance failed")
		}
	}

	// 获取玩家钱包
	w := &wallet.Wallet{}
	if walletIndex < 0 {
		w = nil
	} else {
		w = getWalletByIndex(TonAPI, walletIndex, WalletVersion)
	}

	return cfg, w
}

func GetJettonMetaData() (error, *jetton.MetaData) {
	// 生成jetton minter合约的content
	cfg, err := GetParamsCfg()
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

// GetParamsCfg 获取参数配置
func GetParamsCfg() (*ParamsConfig, error) {
	cfg := &ParamsConfig{}
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

// UpdateParamsCfg 更新参数配置，将配置写入config.json文件
func UpdateParamsCfg(cfg *ParamsConfig) error {
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
