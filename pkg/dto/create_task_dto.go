package dto

import "github.com/flametest/taskd/internal/constant/enum"

type CreatTaskReq struct {
	Body CreateTaskReqBody `json:"body"`
}

type CreateTaskReqBody struct {
	Name       string                 `json:"name" validate:"required,max=255"`
	RefId      string                 `json:"ref_id" validate:"required,max=255"`
	Protocol   enum.Protocol          `json:"protocol"`
	Address    string                 `json:"address" validate:"required,max=500"`
	Params     map[string]interface{} `json:"params"`
	ExecTime   int64                  `json:"exec_time"`
	MaxRetries int                    `json:"max_retries"`
}
