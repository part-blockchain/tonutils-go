package main

import (
	_ "github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// 测试数据
const (
	// MainNetCfg ////////////////////////////////////// 网络配置 ///////////////////////////////////
	MainNetCfg  = "https://ton.org/global.config.json"
	TestNetCfg  = "https://ton.org/testnet-global.config.json"
	MainNetUrl  = "https://toncenter.com/api/v2"
	TestNetUrl  = "https://testnet.toncenter.com/api/v2"
	MainNetScan = "You can view it at: https://tonviewer.com/"
	TestNetScan = "You can view it at: https://testnet.tonviewer.com/"
	// MainNetApiKey / TestNetApiKey ApiKey从·https://t.me/tonapibot·获取
	MainNetApiKey = "f34ef3ba5fd9a2205809cfef6b629026c9a824c394203877a2c309776f2539b3"
	TestNetApiKey = "39b381ff24d91834ee8976a0d6ecec4bf6a74908925ecb4ee2c227ca20b06b5e"

	// Seeds /////////////////////////////////// 钱包配置 /////////////////////////////////////
	Seeds         = "birth pattern then forest walnut then phrase walnut fan pumpkin pattern then cluster blossom verify then forest velvet pond fiction pattern collect then then"
	PriKeyHex     = "073cda7fa5bb8328355e5af63a7f12d359038084db2f20af898b154ea967fb3a1f435378bbed438c427de5c61d3bdfb3f542522877cb2f092efe9723f41bca1f"
	WalletVersion = wallet.V3R2 // 钱包版本
)

// PlayerSeeds 玩家钱包的种子列表, 0: crash game owner address, 1: player address, 2: player address
var PlayerSeeds = []string{
	"birth pattern then forest walnut then phrase walnut fan pumpkin pattern then cluster blossom verify then forest velvet pond fiction pattern collect then then",
	"illegal slot soda divert cliff fantasy cousin injury party gap length era between tortoise network festival near enlist kiss safe gossip few imitate box",
	"denial civil check glide innocent enforce lizard rather impact elder loyal whisper post trap wire lady model pond market range satisfy depth yard arch",
}
