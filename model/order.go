package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/marioarranzr/nefertiti/pricing"
)

type OrderSide int

const (
	ORDER_SIDE_NONE OrderSide = iota
	BUY
	SELL
)

var OrderSideString = map[OrderSide]string{
	ORDER_SIDE_NONE: "",
	BUY:             "buy",
	SELL:            "sell",
}

func (os *OrderSide) String() string {
	return OrderSideString[*os]
}

func NewOrderSide(data string) OrderSide {
	for os := range OrderSideString {
		if os.String() == data {
			return os
		}
	}
	return ORDER_SIDE_NONE
}

func FormatOrderSide(value OrderSide) string {
	if value == BUY {
		return "Buy"
	} else {
		if value == SELL {
			return "Sell"
		}
	}
	return ""
}

type OrderType int

const (
	ORDER_TYPE_NONE OrderType = iota
	LIMIT
	MARKET
)

var OrderTypeString = map[OrderType]string{
	ORDER_TYPE_NONE: "",
	LIMIT:           "limit",
	MARKET:          "market",
}

func (ot *OrderType) String() string {
	return OrderTypeString[*ot]
}

func NewOrderType(data string) OrderType {
	for ot := range OrderTypeString {
		if ot.String() == data {
			return ot
		}
	}
	return ORDER_TYPE_NONE
}

type (
	Order struct {
		Side      OrderSide `json:"-"`
		Market    string    `json:"market"`
		Size      float64   `json:"size"`
		Price     float64   `json:"price"`
		CreatedAt time.Time `json:"createdAt"`
	}
	Orders []Order
)

func (order *Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	return json.Marshal(&struct {
		Side string `json:"side"`
		*Alias
	}{
		Side:  order.Side.String(),
		Alias: (*Alias)(order),
	})
}

func (orders Orders) IndexByPrice(side OrderSide, market string, price float64) int {
	for i, order := range orders {
		if order.Side == side && order.Market == market && order.Price == price {
			return i
		}
	}
	return -1
}

func (orders Orders) OrderByPrice(side OrderSide, market string, price float64) *Order {
	i := orders.IndexByPrice(side, market, price)
	if i != -1 {
		return &orders[i]
	}
	return nil
}

func (orders Orders) Youngest(side OrderSide, def time.Time) time.Time {
	youngest := time.Time{} // January 1, year 1, 00:00:00.000000000 UTC
	for _, order := range orders {
		if order.Side == side {
			if youngest.IsZero() || youngest.Before(order.CreatedAt) {
				youngest = order.CreatedAt
			}
		}
	}
	if youngest.IsZero() {
		return def
	} else {
		return youngest
	}
}

type OrderMetaData struct {
	Nonce int64   `json:"-"`
	Price float64 `json:"p"`
	Mult  float64 `json:"m"`
	Trail bool    `json:"-"`
}

var metaDataNonce int64 = -1

func NewMetaData(price, mult float64) *OrderMetaData {
	return NewMetaDataEx(price, mult, true)
}

func NewMetaDataEx(price, mult float64, trail bool) *OrderMetaData {
	if metaDataNonce == 9 {
		metaDataNonce = 0
	} else {
		metaDataNonce++
	}
	return &OrderMetaData{
		Nonce: metaDataNonce,
		Price: price,
		Mult:  mult,
		Trail: trail,
	}
}

func HasMetaData(str string) bool {
	if len(str) > 1 {
		var (
			err error
			buf []byte
		)
		if str[1] >= '0' && str[1] <= '9' {
			buf, err = base64.RawURLEncoding.DecodeString(str[2:])
		} else {
			buf, err = base64.RawURLEncoding.DecodeString(str[1:])
		}
		if err == nil {
			var omd OrderMetaData
			if json.Unmarshal(buf, &omd) == nil {
				return omd.Price > 0 && omd.Mult > 0
			}
		}
	}
	return false
}

func ParseMetaData(str string) *OrderMetaData {
	var (
		err error
		buf []byte
		out OrderMetaData
	)
	if len(str) > 1 {
		if str[1] >= '0' && str[1] <= '9' {
			out.Trail = str[1] != '0'
			buf, err = base64.RawURLEncoding.DecodeString(str[2:])
		} else {
			out.Trail = true
			buf, err = base64.RawURLEncoding.DecodeString(str[1:])
		}
		if err == nil {
			err = json.Unmarshal(buf, &out)
			if err == nil {
				out.Nonce, err = strconv.ParseInt(str[:1], 10, 64)
			}
		}
	}
	return &out
}

func GetOrderMult(meta string) float64 {
	if len(meta) > 1 {
		var (
			err error
			buf []byte
		)
		if meta[1] >= '0' && meta[1] <= '9' {
			buf, err = base64.RawURLEncoding.DecodeString(meta[2:])
		} else {
			buf, err = base64.RawURLEncoding.DecodeString(meta[1:])
		}
		if err == nil {
			var omd OrderMetaData
			if json.Unmarshal(buf, &omd) == nil {
				if omd.Mult > 0 {
					return omd.Mult
				}
			}
		}
	}
	out := GetMult()
	if out == 0 {
		out = pricing.FIVE_PERCENT
	}
	return out
}

func (md *OrderMetaData) String() string {
	if md.Price == 0 || md.Mult == 0 {
		return ""
	}
	buf, err := json.Marshal(md)
	if err != nil {
		return ""
	}
	if md.Trail {
		return fmt.Sprintf("%d1%s", md.Nonce, base64.RawURLEncoding.EncodeToString(buf))
	} else {
		return fmt.Sprintf("%d0%s", md.Nonce, base64.RawURLEncoding.EncodeToString(buf))
	}
}
