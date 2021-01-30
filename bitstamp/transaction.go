package bitstamp

import (
	"fmt"
	"math"
	"time"

	"github.com/go-errors/errors"
	"github.com/marioarranzr/nefertiti/empty"
)

type (
	Transaction map[string]interface{}
)

func (transaction *Transaction) OrderId() string {
	var out interface{}
	out = (*transaction)["order_id"]
	return empty.AsString(out)
}

func (transaction *Transaction) Market(client *Client) string {
	markets, _ := GetMarkets(client, true)
	for i := range markets {
		price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
		if empty.AsString(price) != "" {
			return markets[i].Name
		}
	}
	return ""
}

func (transaction *Transaction) Price(client *Client) float64 {
	markets, _ := GetMarkets(client, true)
	for i := range markets {
		price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
		if empty.AsString(price) != "" {
			return empty.AsFloat64(price)
		}
	}
	return 0
}

func (transaction *Transaction) Amount(client *Client) float64 {
	markets, _ := GetMarkets(client, true)
	for i := range markets {
		price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
		if empty.AsString(price) != "" {
			return math.Abs(empty.AsFloat64((*transaction)[markets[i].Base]))
		}
	}
	return 0
}

func (transaction *Transaction) Side(client *Client) (string, error) {
	markets, _ := GetMarkets(client, true)
	for i := range markets {
		price := (*transaction)[fmt.Sprintf("%s_%s", markets[i].Base, markets[i].Quote)]
		if empty.AsString(price) != "" {
			quote := empty.AsFloat64((*transaction)[markets[i].Quote])
			if quote < 0 {
				return BUY, nil
			}
			if quote > 0 {
				return SELL, nil
			}
			base := empty.AsFloat64((*transaction)[markets[i].Base])
			if base < 0 {
				return SELL, nil
			}
			if base > 0 {
				return BUY, nil
			}
		}
	}
	return "", errors.Errorf("Unknown transaction side: %+v", transaction)
}

func (transaction *Transaction) DateTime() time.Time {
	var dt interface{}
	dt = (*transaction)["datetime"]
	var out time.Time
	if dt != nil {
		out, _ = time.Parse(TimeFormat, empty.AsString(dt))
	}
	return out
}

type (
	Transactions []Transaction
)

func (transactions Transactions) GetOrders() Transactions {
	var out Transactions
	for _, t := range transactions {
		if t.OrderId() != "" {
			out = append(out, t)
		}
	}
	return out
}

func (transactions Transactions) GetOrdersEx(client *Client, market string) Transactions {
	var out Transactions
	for _, t := range transactions {
		if t.OrderId() != "" {
			if market == "" || t.Market(client) == market {
				out = append(out, t)
			}
		}
	}
	return out
}

func (transactions Transactions) IndexByOrderId(id string) int {
	for i, t := range transactions {
		if t.OrderId() == id {
			return i
		}
	}
	return -1
}
