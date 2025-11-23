package dto

import "github.com/flametest/taskd/internal/constant/enum"

type CreatTaskReq struct {
	Body CreateTaskReqBody `json:"body"`
}

type CreateTaskReqBody struct {
	Name     string        `json:"name"`
	TaskId   string        `json:"task_id"`
	Protocol enum.Protocol `json:"protocol"`
	Address  string        `json:"address"`
	Params   interface{}   `json:"params"`
	ExecTime int64         `json:"exec_time"`
	Retries  int           `json:"retries"`
}
