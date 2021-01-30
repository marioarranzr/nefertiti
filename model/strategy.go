package model

import (
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/pricing"
)

type Strategy int

const (
	STRATEGY_STANDARD                 Strategy = iota // simple. no trailing. no stop-loss.
	STRATEGY_TRAILING                                 // includes trailing. never sells at a loss.
	STRATEGY_TRAILING_STOP_LOSS                       // trailing, but potentially sells at a loss.
	STRATEGY_TRAILING_STOP_LOSS_QUICK                 // trailing, but does not trail forever. sells as soon as ticker >= mult.
	STRATEGY_STOP_LOSS                                // no trailing. potentially sells at a loss. or when ticker >= mult.
)

func GetStrategy() Strategy {
	out := int64(STRATEGY_STANDARD)
	flg := flag.Get("strategy")
	if flg.Exists {
		out, _ = flg.Int64()
	}
	return Strategy(out)
}

func GetMult() float64 {
	out := pricing.FIVE_PERCENT
	flg := flag.Get("mult")
	if flg.Exists {
		out, _ = flg.Float64()
	}
	if out > 0 {
		strategy := GetStrategy()
		// with the trailing strategies, --mult is the distance between the stop price and the ticker price
		if strategy == STRATEGY_TRAILING || strategy == STRATEGY_TRAILING_STOP_LOSS || strategy == STRATEGY_TRAILING_STOP_LOSS_QUICK {
			out = pricing.NewMult(out, 2)
		}
	}
	return out
}
