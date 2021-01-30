package exchanges

import (
	"strings"

	"github.com/go-errors/errors"
	"github.com/marioarranzr/nefertiti/flag"
	"github.com/marioarranzr/nefertiti/model"
	"github.com/marioarranzr/nefertiti/passphrase"
)

type Exchanges []model.Exchange

func (exchanges *Exchanges) FindByName(name string) model.Exchange {
	for _, exchange := range *exchanges {
		if exchange.GetInfo().Equals(name) {
			return exchange
		}
	}
	return nil
}

func New() *Exchanges {
	var out Exchanges
	out = append(out, NewBinance())
	return &out
}

func getPrecFromStr(value string, def int) int {
	i := strings.Index(value, ".")
	if i > -1 {
		n := i + 1
		for n < len(value) {
			if string(value[n]) != "0" {
				return n - i
			}
			n++
		}
		return 0
	}
	return def
}

func promptForApiKeys(exchange string) (apiKey, apiSecret string, err error) {
	apiKey = flag.Get("api-key").String()
	if apiKey == "" {
		if flag.Listen() {
			return "", "", errors.New("missing argument: api-key")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API key"); err != nil {
			return "", "", errors.Wrap(err, 1)
		}
		apiKey = string(data)
	}

	apiSecret = flag.Get("api-secret").String()
	if apiSecret == "" {
		if flag.Listen() {
			return "", "", errors.New("missing argument: api-secret")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API secret"); err != nil {
			return "", "", errors.Wrap(err, 1)
		}
		apiSecret = string(data)
	}

	return apiKey, apiSecret, nil
}

func promptForApiKeysEx(exchange string) (apiKey, apiSecret, apiPassphrase string, err error) {
	apiKey, apiSecret, err = promptForApiKeys(exchange)

	if err != nil {
		return apiKey, apiSecret, "", err
	}

	apiPassphrase = flag.Get("api-passphrase").String()
	if apiPassphrase == "" {
		if flag.Listen() {
			return "", "", "", errors.New("missing argument: api-passphrase")
		}
		var data []byte
		if data, err = passphrase.Read(exchange + " API passphrase"); err != nil {
			return "", "", "", errors.Wrap(err, 1)
		}
		apiPassphrase = string(data)
	}

	return apiKey, apiSecret, apiPassphrase, nil
}
