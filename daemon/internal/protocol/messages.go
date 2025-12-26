package protocol

type MouseMoveMessage struct {
	Type string `json:"type"`
	Dx   int32  `json:"dx"`
	Dy   int32  `json:"dy"`
}
