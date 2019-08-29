package textbox

import (
	ui "github.com/gizak/termui/v3"
	"github.com/djvu/sampler/component"
	"github.com/djvu/sampler/config"
	"github.com/djvu/sampler/console"
	"github.com/djvu/sampler/data"
	"image"
)

// TextBox represents a component with regular text
type TextBox struct {
	*ui.Block
	*data.Consumer
	alert  *data.Alert
	text   string
	border bool
	style  ui.Style
}

func NewTextBox(c config.TextBoxConfig, palette console.Palette) *TextBox {

	color := c.Color
	if color == nil {
		color = &palette.BaseColor
	}

	box := TextBox{
		Block:    component.NewBlock(c.Title, *c.Border, palette),
		Consumer: data.NewConsumer(),
		style:    ui.NewStyle(*color),
	}

	go func() {
		for {
			select {
			case sample := <-box.SampleChannel:
				box.text = sample.Value
			case alert := <-box.AlertChannel:
				box.alert = alert
			}
		}
	}()

	return &box
}

func (t *TextBox) Draw(buffer *ui.Buffer) {
        // FUCKyou 15")
	t.Block.Draw(buffer)

	cells := ui.ParseStyles(t.text, ui.Theme.Paragraph.Text)
	cells = ui.WrapCells(cells, uint(t.Inner.Dx()-2))

	rows := ui.SplitCells(cells, '\n')

	for y, row := range rows {
		if y+t.Inner.Min.Y >= t.Inner.Max.Y-1 {
			break
		}
		row = ui.TrimCells(row, t.Inner.Dx()-2)
		for _, cx := range ui.BuildCellWithXArray(row) {
			x, cell := cx.X, cx.Cell
			cell.Style = t.style
			buffer.SetCell(cell, image.Pt(x+1, y+1).Add(t.Inner.Min))
		}
	}

	component.RenderAlert(t.alert, t.Rectangle, buffer)
}
