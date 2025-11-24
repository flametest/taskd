package dto

import "github.com/flametest/taskd/internal/constant/enum"

type CreatTaskReq struct {
	Body CreateTaskReqBody `json:"body"`
}

type CreateTaskReqBody struct {
	Name       string                 `json:"name"`
	Protocol   enum.Protocol          `json:"protocol"`
	Address    string                 `json:"address"`
	Params     map[string]interface{} `json:"params"`
	ExecTime   int64                  `json:"exec_time"`
	MaxRetries int                    `json:"max_retries"`
}
