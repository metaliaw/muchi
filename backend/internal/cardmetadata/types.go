package cardmetadata

type Request struct {
	Name     string
	Language string
	Edition  string
	Foil     bool
}

type Metadata struct {
	Name        string `json:"name"`
	PrintedName string `json:"printed_name,omitempty"`
	Edition     string `json:"edition,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Print struct {
	Edition         string
	EditionName     string
	CollectorNumber string
	Image           string
}
