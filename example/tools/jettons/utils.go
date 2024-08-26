package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
	"strings"
)

// GenWalletByMnemonicWords Generate Wallet based on mnemonic words
func GenWalletByMnemonicWords(api wallet.TonAPI, seeds string, version wallet.Version) (error, *wallet.Wallet) {
	log.Println("generate address by mnemonic words...")
	// seed words of account, you can generate them with any wallet or using wallet.NewSeed() method
	words := strings.Split(seeds, " ")
	w, err := wallet.FromSeed(api, words, version)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return err, nil
	}

	return nil, w
}
func getNFTCollectionCode() *cell.Cell {
	var hexBOC = "b5ee9c724102140100021f000114ff00f4a413f4bcf2c80b0102016202030202cd04050201200e0f04e7d10638048adf000e8698180b8d848adf07d201800e98fe99ff6a2687d20699fea6a6a184108349e9ca829405d47141baf8280e8410854658056b84008646582a802e78b127d010a65b509e58fe59f80e78b64c0207d80701b28b9e382f970c892e000f18112e001718112e001f181181981e0024060708090201200a0b00603502d33f5313bbf2e1925313ba01fa00d43028103459f0068e1201a44343c85005cf1613cb3fccccccc9ed54925f05e200a6357003d4308e378040f4966fa5208e2906a4208100fabe93f2c18fde81019321a05325bbf2f402fa00d43022544b30f00623ba9302a402de04926c21e2b3e6303250444313c85005cf1613cb3fccccccc9ed54002c323401fa40304144c85005cf1613cb3fccccccc9ed54003c8e15d4d43010344130c85005cf1613cb3fccccccc9ed54e05f04840ff2f00201200c0d003d45af0047021f005778018c8cb0558cf165004fa0213cb6b12ccccc971fb008002d007232cffe0a33c5b25c083232c044fd003d0032c03260001b3e401d3232c084b281f2fff2742002012010110025bc82df6a2687d20699fea6a6a182de86a182c40043b8b5d31ed44d0fa40d33fd4d4d43010245f04d0d431d430d071c8cb0701cf16ccc980201201213002fb5dafda89a1f481a67fa9a9a860d883a1a61fa61ff480610002db4f47da89a1f481a67fa9a9a86028be09e008e003e00b01a500c6e"
	codeCellBytes, _ := hex.DecodeString(hexBOC)

	codeCell, err := cell.FromBOC(codeCellBytes)
	if err != nil {
		panic(err)
	}

	return codeCell
}

func getNFTItemCode() *cell.Cell {
	var hexBOC = "b5ee9c7241020d010001d0000114ff00f4a413f4bcf2c80b0102016202030202ce04050009a11f9fe00502012006070201200b0c02d70c8871c02497c0f83434c0c05c6c2497c0f83e903e900c7e800c5c75c87e800c7e800c3c00812ce3850c1b088d148cb1c17cb865407e90350c0408fc00f801b4c7f4cfe08417f30f45148c2ea3a1cc840dd78c9004f80c0d0d0d4d60840bf2c9a884aeb8c097c12103fcbc20080900113e910c1c2ebcb8536001f65135c705f2e191fa4021f001fa40d20031fa00820afaf0801ba121945315a0a1de22d70b01c300209206a19136e220c2fff2e192218e3e821005138d91c85009cf16500bcf16712449145446a0708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb00104794102a375be20a00727082108b77173505c8cbff5004cf1610248040708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb000082028e3526f0018210d53276db103744006d71708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb0093303234e25502f003003b3b513434cffe900835d27080269fc07e90350c04090408f80c1c165b5b60001d00f232cfd633c58073c5b3327b5520bf75041b"
	codeCellBytes, _ := hex.DecodeString(hexBOC)

	codeCell, err := cell.FromBOC(codeCellBytes)
	if err != nil {
		panic(err)
	}

	return codeCell
}

func getContractData(collectionOwnerAddr, royaltyAddr *address.Address) *cell.Cell {
	// storage schema
	// default#_ royalty_factor:uint16 royalty_base:uint16 royalty_address:MsgAddress = RoyaltyParams;
	// storage#_ owner_address:MsgAddress next_item_index:uint64
	//           ^[collection_content:^Cell common_content:^Cell]
	//           nft_item_code:^Cell
	//           royalty_params:^RoyaltyParams
	//           = Storage;

	royalty := cell.BeginCell().
		MustStoreUInt(5, 16). // 5% royalty
		MustStoreUInt(100, 16). // denominator
		MustStoreAddr(royaltyAddr). // fee addr destination
		EndCell()

	// collection data
	collectionContent := nft.ContentOffchain{URI: "https://tonutils.com/collection.json"}
	collectionContentCell, _ := collectionContent.ContentCell()

	// prefix for NFTs data
	uri := "https://tonutils.com/nft/"
	commonContentCell := cell.BeginCell().MustStoreStringSnake(uri).EndCell()

	contentRef := cell.BeginCell().
		MustStoreRef(collectionContentCell).
		MustStoreRef(commonContentCell).
		EndCell()

	data := cell.BeginCell().MustStoreAddr(collectionOwnerAddr).
		MustStoreUInt(0, 64).
		MustStoreRef(contentRef).
		MustStoreRef(getNFTItemCode()).
		MustStoreRef(royalty).
		EndCell()

	return data
}

// DeployJettonMinter 部署Jetton Minter合约
func DeployJettonMinter(filePath string) error {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed")
		}
	}
	log.Println("Deploying jetton contract to ton blockchain...")
	err, w := GenWalletByMnemonicWords(TonAPI, Seeds, WalletVersion)
	if err != nil || nil == w {
		return errors.New("generate wallet by seed words failed")
	}

	log.Println("Deploy wallet:", w.WalletAddress().String())

	msgBody := cell.BeginCell().EndCell()

	fmt.Println("Deploying NFT collection contract to mainnet...")
	addr, _, _, err := w.DeployContractWaitTransaction(context.Background(), tlb.MustFromTON("0.05"),
		msgBody, getNFTCollectionCode(), getContractData(w.WalletAddress(), w.WalletAddress()))
	if err != nil {
		panic(err)
	}

	fmt.Println("Deployed contract addr:", addr.String())
	return nil
}

// NewJettonMasterClient 创建Jetton Master Client
func NewJettonMasterClient(jettonMinterAddr string) (error, *context.Context, *jetton.Client) {
	if nil == TonAPI {
		TonAPI = GetTonAPIIns()
		if nil == TonAPI {
			return errors.New("get ton api instance failed"), nil, nil
		}
	}
	client := TonAPI.Client()
	// bound all requests to single ton node
	ctx := client.StickyContext(context.Background())
	tokenContract := address.MustParseAddr(jettonMinterAddr)
	master := jetton.NewJettonMasterClient(TonAPI, tokenContract)
	return nil, &ctx, master
}

// GetJettonData 获取Jetton Token信息
func GetJettonData(jettonMinterAddr string) (error, *jetton.Data) {
	err, ctx, master := NewJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == ctx {
		return errors.New("new jetton master client failed"), nil
	}
	data, err := master.GetJettonData(*ctx)
	if err != nil {
		log.Fatal(err)
	}

	content := data.Content.(*jetton.JettonMetaData)
	log.Println("total supply:", data.TotalSupply.Uint64())
	log.Println("mintable:", data.Mintable)
	log.Println("admin addr:", data.AdminAddr)
	log.Println("jetton content:")
	log.Println("	name:", content.Name)
	log.Println("	Description:", content.Description)
	log.Println("	Symbol:", content.Symbol)
	log.Println("	Decimals:", content.Decimals)
	log.Println("	Image:", content.Image)
	log.Println("	URI:", content.URI) // 链下
	log.Println("	AmountType:", content.AmountType)
	log.Println("	RenderType:", content.RenderType)

	return nil, data
}

// GetJettonWallet 获取Jetton Wallet Data信息
func GetJettonWallet(jettonMinterAddr, ownerAddr string) error {
	err, pCtx, master := NewJettonMasterClient(jettonMinterAddr)
	if err != nil || nil == master || nil == pCtx {
		return errors.New("new jetton master client failed")
	}
	ctx := *pCtx
	tokenWallet, err := master.GetJettonWallet(ctx, address.MustParseAddr(ownerAddr))
	if err != nil {
		log.Fatal(err)
	}
	// 获取jetton wallet data
	jettonWalletData, err := tokenWallet.GetWalletData(ctx)
	if err != nil {
		log.Fatal(err)
	}
	decimals := 6
	log.Println("jetton minter address:", jettonWalletData.JettonMinterAddr)
	log.Println("jetton wallet owner address:", jettonWalletData.OwnerAddr)
	log.Println("jetton balance:", tlb.MustFromNano(jettonWalletData.Balance, decimals))

	return nil
}
