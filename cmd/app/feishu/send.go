package feishu

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type options struct {
	token string
	sign  string
	msg   string
	file  string
}

func newSendCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "send",
		Short: "For sending simple message to feishu webhook",
		RunE: func(_ *cobra.Command, _ []string) error {
			if len(o.file) > 0 {
				b, err := ioutil.ReadFile(o.file)
				if err != nil {
					return err
				}
				o.msg = string(b)
			} else if o.msg == "-" {
				var lines []string
				for {
					rd := bufio.NewReader(os.Stdin)
					buf, err := rd.ReadString('\n')
					if err != nil {
						return err
					}
					if buf[0] == '\n' {
						break
					}
					lines = append(lines, string(buf))
				}
				o.msg = strings.Join(lines, "")
			}
			if len(o.msg) == 0 {
				return nil
			}
			return send(o.token, o.sign, strings.ReplaceAll(o.msg, "\\n", "\n"))
		},
	}
	cmd.Flags().StringVar(&o.token, "token", "", "feishu webhook token")
	cmd.Flags().StringVar(&o.sign, "sign", "", "feishu webhook signature")
	cmd.Flags().StringVar(&o.msg, "msg", "", "message to send, default scan input from stdin")
	cmd.Flags().StringVar(&o.file, "file", "", "message to send in file")

	cmd.MarkFlagRequired("token")
	return cmd
}

const webhookURI = "https://open.feishu.cn/open-apis/bot/v2/hook/%s"

func genSign(secret string, timestamp int64) (string, error) {
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

type content struct {
	Text string `json:"text"`
}

type payload struct {
	Timestamp string  `json:"timestamp,omitempty"`
	Sign      string  `json:"sign,omitempty"`
	MsgType   string  `json:"msg_type"`
	Content   content `json:"content,omitempty"`
}

func send(token string, sign string, msg string) (err error) {
	pl := &payload{
		MsgType: "text",
		Content: content{msg},
	}
	if len(sign) > 0 {
		now := time.Now().Unix()
		pl.Timestamp = strconv.FormatInt(now, 10)
		pl.Sign, err = genSign(sign, now)
		if err != nil {
			return
		}
	}
	b, err := json.Marshal(pl)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf(webhookURI, token), "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 >= 4 {
		respContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected status(%d): %v", resp.StatusCode, string(respContent))
	}
	return nil
}
