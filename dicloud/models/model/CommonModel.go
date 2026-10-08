package model

const (
	GATEWAY          = "gateway"
	USER             = "user"
	STREAM           = "stream"
	USERNAME_PATTERN = `^[a-zA-Z][a-zA-Z0-9_]{4,15}$`
	PHONT_PATTERN    = `^(13[0-9]|14[0-9]|15[0-9]|166|17[0-9]|18[0-9]|19[8|9])\d{8}$`
)

type DataValue struct {
	Data  string  `json:"data"`
	Value float64 `json:"value"`
}

type Page struct {
	Pagenum     int64  `json:"pagenum"`
	Pagesize    int64  `json:"pagesize"`
	Ordervalue  int    `json:"ordervalue"`
	Orderobject string `json:"orderobject"`
}

type Date struct {
	Datestart uint64 `json:"datestart"`
	Dateend   uint64 `json:"dateend"`
}
