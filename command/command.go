package command

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/marioarranzr/nefertiti/flag"
)

type CommandCallBack func(err error)

type CommandMeta struct {
	Port       *int64
	AppName    string
	AppVersion string
	CallBack   *CommandCallBack
}

func (cm *CommandMeta) ReturnError(err error) int {
	// step #1: execute the callback function (passing the error back to main)
	var cb CommandCallBack
	cb = *cm.CallBack
	cb(err)
	// step #2: return 1 as an error code
	return 1
}

func (cm *CommandMeta) ReturnSuccess() error {
	if flag.Listen() {
		flg := flag.Get("hub")
		if flg.Exists {
			hub, err := flg.Int64()
			if err != nil {
				return fmt.Errorf("hub %v is invalid", flg)
			}
			data := url.Values{}
			data.Set("port", strconv.FormatInt(*cm.Port, 10))
			req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/callback", hub), strings.NewReader(data.Encode()))
			if err != nil {
				return err
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			if _, err := http.DefaultClient.Do(req); err != nil {
				return err
			}
		}
	}
	return nil
}
