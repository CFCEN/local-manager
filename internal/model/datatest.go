package model

type DataTestPing struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type DataTestTableCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DataTestSummary struct {
	OK     bool                 `json:"ok"`
	Tables []DataTestTableCount `json:"tables"`
}

type DataTestEchoRequest struct {
	Message string `json:"message"`
}

type DataTestEchoResponse struct {
	Message string `json:"message"`
}
