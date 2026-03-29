package analyzer

type Vector3 struct {
	X float64 `json:"x" xml:"x,attr"`
	Y float64 `json:"y" xml:"y,attr"`
	Z float64 `json:"z" xml:"z,attr"`
}

type componentInfo struct {
	parent string
	pos    *Vector3
	class  string
	macro  string
	code   string
}

type ScanResult struct {
	ID          string  `json:"id"`
	Type        string  `json:"type,omitempty"`
	Name        string  `json:"name,omitempty"`
	Class       string  `json:"class,omitempty"`
	Macro       string  `json:"macro,omitempty"`
	Owner       string  `json:"owner,omitempty"`
	Code        string  `json:"code,omitempty"`
	IsWreck     bool    `json:"is_wreck"`
	IsDecrypted bool    `json:"is_decrypted,omitempty"`
	Sector      string  `json:"sector"`
	System      string  `json:"system"`
	Pos         Vector3 `json:"pos"`
}

type AnalysisResults struct {
	Info    map[string]string `json:"info"`
	Khaak   []ScanResult      `json:"khaak"`
	Ships   []ScanResult      `json:"ships"`
	Unowned []ScanResult      `json:"unowned"`
	Vaults  []ScanResult      `json:"vaults"`
}

func NewAnalysisResults() *AnalysisResults {
	return &AnalysisResults{
		Info:    make(map[string]string),
		Khaak:   make([]ScanResult, 0),
		Ships:   make([]ScanResult, 0),
		Unowned: make([]ScanResult, 0),
		Vaults:  make([]ScanResult, 0),
	}
}
