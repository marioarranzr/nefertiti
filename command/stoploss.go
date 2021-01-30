package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marioarranzr/nefertiti/exchanges"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
)

type (
	StopLossCommand struct {
		*CommandMeta
	}
)

func (c *StopLossCommand) Run(args []string) int {
	var (
		err error
		flg *flag.Flag
	)

	sandbox := false
	flg = flag.Get("sandbox")
	if flg.Exists {
		sandbox = flg.String() == "Y"
	}

	flg = flag.Get("exchange")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: exchange"))
	}
	exchange := exchanges.New().FindByName(flg.String())
	if exchange == nil {
		return c.ReturnError(fmt.Errorf("exchange %v does not exist", flg))
	}

	var kind model.OrderType = model.LIMIT
	flg = flag.Get("type")
	if flg.Exists {
		kind = model.NewOrderType(flg.String())
		if kind == model.ORDER_TYPE_NONE {
			return c.ReturnError(fmt.Errorf("type %v is invalid", flg))
		}
	}

	var markets []model.Market
	if markets, err = exchange.GetMarkets(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	flg = flag.Get("market")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: market"))
	}
	market := flg.String()
	if model.HasMarket(markets, market) == false {
		return c.ReturnError(fmt.Errorf("market %s does not exist", market))
	}

	var size float64
	flg = flag.Get("size")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: size"))
	}
	if size, err = flg.Float64(); err != nil {
		return c.ReturnError(fmt.Errorf("size %v is invalid", flg))
	}

	var price float64 = 0
	flg = flag.Get("price")
	if flg.Exists {
		if price, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("price %v is invalid", flg))
		}
	} else if kind == model.LIMIT {
		return c.ReturnError(errors.New("missing argument: price"))
	}

	var client interface{}
	if client, err = exchange.GetClient(true, sandbox); err != nil {
		return c.ReturnError(err)
	}

	var out []byte
	if out, err = exchange.StopLoss(
		client,
		market,
		size,
		price,
		kind, "",
	); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(string(out))

	return 0
}

func (c *StopLossCommand) Help() string {
	text := `
Usage: ./cryptotrader stoploss [options]

The stoploss command places a stop-loss order with the specified exchange.

Options:
  --exchange = name
  --type     = [limit|market]
  --market   = a valid market pair
  --size     = amount of cryptocurrency to sell
  --price    = price per unit
`
	return strings.TrimSpace(text)
}

func (c *StopLossCommand) Synopsis() string {
	return "Place a stop-loss order with the specified exchange."
}
