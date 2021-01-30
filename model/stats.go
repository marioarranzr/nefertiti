package model

import "github.com/marioarranzr/nefertiti/pricing"

type Stats struct {
	Market    string
	High      float64
	Low       float64
	BtcVolume float64
}

func (s *Stats) Avg(exchange Exchange, sandbox bool) (float64, error) {
	client, err := exchange.GetClient(false, sandbox)
	if err != nil {
		return 0, err
	}
	prec, err := exchange.GetPricePrec(client, s.Market)
	if err != nil {
		return 0, err
	}
	return pricing.RoundToPrecision(((s.High + s.Low) / 2), prec), nil
}
