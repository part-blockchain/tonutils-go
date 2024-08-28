package jetton

import (
	"crypto/sha256"
	"fmt"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

type ContentAny interface {
	ContentCell() (*cell.Cell, error)
}

type ContentOffchain struct {
	URI string
}

type ContentOnchain struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	ImageData   string `json:"image_data"`
	Symbol      string `json:"symbol"`
	Decimals    string `json:"decimals"`
	AmountStyle string `json:"amount_style"`
	RenderType  string `json:"render_type"`

	attributes *cell.Dictionary
}

type MetaData struct {
	ContentOffchain `json:"offchain"`
	ContentOnchain  `json:"onchain"`
}

func GetContentFromCell(c *cell.Cell) (ContentAny, error) {
	return GetContentFromSlice(c.BeginParse())
}

func GetContentFromSlice(s *cell.Slice) (ContentAny, error) {
	if s.BitsLeft() < 8 {
		if s.RefsNum() == 0 {
			return &ContentOffchain{}, nil
		}
		s = s.MustLoadRef()
	}
	// 取slice的前8位，判断是链下数据（URI）， 还是包含链上数据的字典
	typ, err := s.LoadUInt(8)
	if err != nil {
		return nil, fmt.Errorf("failed to load type: %w", err)
	}
	t := uint8(typ)

	switch t {
	case 0x00:
		// 包含链上数据，加载字典类型
		dict, err := s.LoadDict(256)
		if err != nil {
			return nil, fmt.Errorf("failed to load dict onchain data: %w", err)
		}

		uri := string(getOnchainVal(dict, "uri"))

		on := ContentOnchain{
			Name:        string(getOnchainVal(dict, "name")),
			Description: string(getOnchainVal(dict, "description")),
			Image:       string(getOnchainVal(dict, "image")),
			ImageData:   string(getOnchainVal(dict, "image_data")),
			// attributes:  dict,
			Symbol:      string(getOnchainVal(dict, "symbol")),
			Decimals:    string(getOnchainVal(dict, "decimals")),
			AmountStyle: string(getOnchainVal(dict, "amount_style")),
			RenderType:  string(getOnchainVal(dict, "render_type")),
		}

		var content ContentAny

		if uri != "" {
			content = &MetaData{
				ContentOffchain: ContentOffchain{
					URI: uri,
				},
				ContentOnchain: on,
			}
		} else {
			content = &on
		}

		return content, nil
	case 0x01:
		str, err := s.LoadStringSnake()
		if err != nil {
			return nil, fmt.Errorf("failed to load snake offchain data: %w", err)
		}

		return &ContentOffchain{
			URI: str,
		}, nil
	default:
		str, err := s.LoadStringSnake()
		if err != nil {
			return nil, fmt.Errorf("failed to load snake offchain data: %w", err)
		}

		return &ContentOffchain{
			URI: string(t) + str,
		}, nil
	}
}

func getOnchainVal(dict *cell.Dictionary, key string) []byte {
	h := sha256.New()
	h.Write([]byte(key))

	val := dict.Get(cell.BeginCell().MustStoreSlice(h.Sum(nil), 256).EndCell())
	if val != nil {
		v, err := val.BeginParse().LoadRef()
		if err != nil {
			return nil
		}

		typ, err := v.LoadUInt(8)
		if err != nil {
			return nil
		}

		switch typ {
		case 0x01:
			// TODO: add support for chunked
			return nil
		default:
			data, _ := v.LoadBinarySnake()
			return data
		}
	}

	return nil
}

func setOnchainVal(dict *cell.Dictionary, key string, val []byte) error {
	h := sha256.New()
	h.Write([]byte(key))

	v := cell.BeginCell().MustStoreUInt(0x00, 8)
	if err := v.StoreBinarySnake(val); err != nil {
		return err
	}

	err := dict.Set(cell.BeginCell().MustStoreSlice(h.Sum(nil), 256).EndCell(), cell.BeginCell().MustStoreRef(v.EndCell()).EndCell())
	if err != nil {
		return err
	}

	return nil
}

func (c *ContentOffchain) ContentCell() (*cell.Cell, error) {
	return cell.BeginCell().MustStoreUInt(0x01, 8).MustStoreStringSnake(c.URI).EndCell(), nil
}

func (c *MetaData) ContentCell() (*cell.Cell, error) {
	if c.attributes == nil {
		c.attributes = cell.NewDict(256)
	}

	// 生成链下数据的cell
	if c.URI != "" && getOnchainVal(c.attributes, "uri") == nil {
		ci := cell.BeginCell()

		err := ci.StoreStringSnake(c.URI)
		if err != nil {
			return nil, err
		}

		err = setOnchainVal(c.attributes, "uri", []byte(c.URI))
		if err != nil {
			return nil, err
		}
	}
	// 生成链上数据的cell
	return c.ContentOnchain.ContentCell()
}

func (c *ContentOnchain) SetAttribute(name, value string) error {
	return c.SetAttributeBinary(name, []byte(value))
}

func (c *ContentOnchain) SetAttributeBinary(name string, value []byte) error {
	if c.attributes == nil {
		c.attributes = cell.NewDict(256)
	}

	err := setOnchainVal(c.attributes, name, value)
	if err != nil {
		return fmt.Errorf("failed to set attribute: %w", err)
	}
	return nil
}

func (c *ContentOnchain) SetAttributeCell(key string, cl *cell.Cell) error {
	if c.attributes == nil {
		c.attributes = cell.NewDict(256)
	}

	h := sha256.New()
	h.Write([]byte(key))

	err := c.attributes.Set(cell.BeginCell().MustStoreSlice(h.Sum(nil), 256).EndCell(), cell.BeginCell().MustStoreRef(cl).EndCell())
	if err != nil {
		return err
	}

	return nil
}

func (c *ContentOnchain) GetAttribute(name string) string {
	return string(c.GetAttributeBinary(name))
}

func (c *ContentOnchain) GetAttributeBinary(name string) []byte {
	return getOnchainVal(c.attributes, name)
}

// ContentCell 生成链上数据的cell
func (c *ContentOnchain) ContentCell() (*cell.Cell, error) {
	if c.attributes == nil {
		c.attributes = cell.NewDict(256)
	}

	if len(c.Name) > 0 {
		err := setOnchainVal(c.attributes, "name", []byte(c.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to store name: %w", err)
		}
	}
	if len(c.Description) > 0 {
		err := setOnchainVal(c.attributes, "description", []byte(c.Description))
		if err != nil {
			return nil, fmt.Errorf("failed to store description: %w", err)
		}
	}
	if len(c.Image) > 0 {
		err := setOnchainVal(c.attributes, "image", []byte(c.Image))
		if err != nil {
			return nil, fmt.Errorf("failed to store image: %w", err)
		}
	}
	if len(c.ImageData) > 0 {
		err := setOnchainVal(c.attributes, "image_data", []byte(c.ImageData))
		if err != nil {
			return nil, fmt.Errorf("failed to store image_data: %w", err)
		}
	}
	if len(c.Symbol) > 0 {
		err := setOnchainVal(c.attributes, "symbol", []byte(c.Symbol))
		if err != nil {
			return nil, fmt.Errorf("failed to store symbol: %w", err)
		}
	}
	if len(c.Decimals) > 0 {
		err := setOnchainVal(c.attributes, "decimals", []byte(c.Decimals))
		if err != nil {
			return nil, fmt.Errorf("failed to store decimals: %w", err)
		}
	}

	if len(c.AmountStyle) > 0 {
		err := setOnchainVal(c.attributes, "amount_style", []byte(c.AmountStyle))
		if err != nil {
			return nil, fmt.Errorf("failed to store amount_style: %w", err)
		}
	}
	if len(c.RenderType) > 0 {
		err := setOnchainVal(c.attributes, "render_type", []byte(c.RenderType))
		if err != nil {
			return nil, fmt.Errorf("failed to store render_type: %w", err)
		}
	}

	return cell.BeginCell().MustStoreUInt(0x00, 8).MustStoreDict(c.attributes).EndCell(), nil
}

// GenJettonContentCell 生成jetton content的cell
func GenJettonContentCell(content ContentAny) (*cell.Cell, error) {
	if content == nil {
		return cell.BeginCell().EndCell(), nil
	}
	// 链下参数
	if off, ok := content.(*ContentOffchain); ok {
		// https://github.com/ton-blockchain/TIPs/issues/64
		// Standard says that prefix should be 0x01, but looks like it was misunderstanding in other implementations and 0x01 was dropped
		// so, we make compatibility
		return cell.BeginCell().MustStoreStringSnake(off.URI).EndCell(), nil
	}
	return content.ContentCell()
}
