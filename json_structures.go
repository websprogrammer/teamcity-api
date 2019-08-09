package main

type Builds struct {
	Build []Build `json:"build"`
}

type Build struct {
	Number     string  `json:"number,attr"`
	Id         int64   `json:"id,attr"`
	Status     string  `json:"status,attr"`
	State      string  `json:"state,attr"`
	StatusText string  `json:"statusText"`
	FinishDate string  `json:"finishDate"`
	Changes    Changes `json:"changes"`
}

type Changes struct {
	Change []Change `json:"change"`
}

type Change struct {
	UserName string `json:"username,attr"`
	Comment  string `json:"comment"`
}

type SocketData struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
}

type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
}
