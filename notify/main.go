package notify

import (
	"github.com/marioarranzr/nefertiti/model"
)

type Services []model.Notify

func (services *Services) Init(interactive, verify bool) (model.Notify, error) {
	for _, service := range *services {
		ok, err := service.PromptForKeys(interactive, verify)
		if ok {
			return service, err
		}
	}
	return nil, nil
}

func New() *Services {
	var out Services
	out = append(out, NewPushover())
	out = append(out, NewTelegram())
	return &out
}
