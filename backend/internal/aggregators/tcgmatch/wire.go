package tcgmatch

type catalogReply struct {
	Products []catalogProduct `json:"products"`
}

type catalogProduct struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	TCG       string   `json:"tcg"`
	Type      string   `json:"type"`
	Image     string   `json:"image"`
	SetID     string   `json:"setId"`
	SetCode   string   `json:"setCode"`
	SetName   string   `json:"setName"`
	CardCode  string   `json:"cardCode"`
	Languages []string `json:"languages"`
	Function  string   `json:"-"`
}

type listingsReply struct {
	Success bool      `json:"success"`
	Data    []listing `json:"data"`
}

type listing struct {
	ID            string `json:"_id"`
	Name          string `json:"name"`
	TCG           string `json:"tcg"`
	Language      string `json:"language"`
	Status        string `json:"status"`
	Quantity      int    `json:"quantity"`
	Price         int64  `json:"price"`
	IsActive      bool   `json:"isActive"`
	IsHolo        bool   `json:"isHolo"`
	IsHoloReverse bool   `json:"isHoloReverse"`
	User          struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
}
