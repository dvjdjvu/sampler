package component

import (
	ui "github.com/gizak/termui/v3"
	//"github.com/djvu/sampler/component/util"
	"github.com/djvu/sampler/console"
)

type NagWindow struct {
	*ui.Block
	palette  console.Palette
	accepted bool
}

func NewNagWindow(palette console.Palette) *NagWindow {
	return &NagWindow{
		Block:    NewBlock("", false, palette),
		palette:  palette,
		accepted: false,
	}
}

func (n *NagWindow) Accept() {
	n.accepted = true
}

func (n *NagWindow) IsAccepted() bool {
	return n.accepted
}

func (n *NagWindow) Draw(buffer *ui.Buffer) {
        return
}
