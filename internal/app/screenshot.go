package app

type ScreenshotFrame struct {
	Image    string          `json:"image"`
	Width    int             `json:"width"`
	Height   int             `json:"height"`
	Added    bool            `json:"added"`
	WorkArea *ScreenshotRect `json:"workArea,omitempty"`
}

type ScreenshotRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
