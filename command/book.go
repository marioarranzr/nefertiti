package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/marioarranzr/nefertiti/exchanges"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
)

type (
	BookCommand struct {
		*CommandMeta
	}
)

func (c *BookCommand) Run(args []string) int {
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

	var side model.BookSide
	flg = flag.Get("side")
	if flg.String() == "asks" {
		side = model.BOOK_SIDE_ASKS
	} else {
		side = model.BOOK_SIDE_BIDS
	}

	var agg float64
	flg = flag.Get("agg")
	if flg.Exists == false {
		return c.ReturnError(errors.New("missing argument: agg"))
	} else {
		if agg, err = flg.Float64(); err != nil {
			return c.ReturnError(fmt.Errorf("agg value %v is invalid", flg))
		}
	}

	var client interface{}
	if client, err = exchange.GetClient(false, sandbox); err != nil {
		return c.ReturnError(err)
	}

	var book1 interface{}
	if book1, err = exchange.GetBook(client, market, side); err != nil {
		return c.ReturnError(err)
	}

	var book2 model.Book
	if book2, err = exchange.Aggregate(client, book1, market, agg); err != nil {
		return c.ReturnError(err)
	}

	// sort the order book by size (highest order size first)
	sort.Slice(book2, func(i1, i2 int) bool {
		return book2[i1].Size > book2[i2].Size
	})

	var out []byte
	if out, err = json.Marshal(book2); err != nil {
		return c.ReturnError(err)
	}

	fmt.Println(string(out))

	return 0
}

func (c *BookCommand) Help() string {
	text := `
Usage: ./cryptotrader book [options]

The book command returns a list of all public orders on a market.

Options:
  --exchange = name
  --market   = a valid market pair
  --side     = [bids|asks] (optional, defaults to bids)
  --agg      = aggregate the book to nearest multiple of agg
`
	return strings.TrimSpace(text)
}

func (c *BookCommand) Synopsis() string {
	return "Get a list of all public orders on a market."
}
