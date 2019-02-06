package widgets

import (
	"fmt"
	"github.com/sqshq/sampler/console"
	"github.com/sqshq/sampler/data"
	"image"
	"math"
	"strconv"
	"sync"
	"time"

	ui "github.com/sqshq/termui"
)

// TODO split into runchart, grid, legend files
const (
	xAxisLegendWidth  = 20
	xAxisLabelsHeight = 1
	xAxisLabelsWidth  = 8
	xAxisLabelsGap    = 2
	xAxisGridWidth    = xAxisLabelsGap + xAxisLabelsWidth

	yAxisLabelsHeight = 1
	yAxisLabelsGap    = 1

	chartHistoryReserve = 5
)

type RunChart struct {
	ui.Block
	lines     []timeLine
	grid      chartGrid
	precision int
	timescale time.Duration
	mutex     *sync.Mutex
}

type chartGrid struct {
	valueExtrema valueExtrema
	timeRange    timeRange
	linesCount   int
	paddingWidth int
	maxTimeWidth int
	minTimeWidth int
}

type timePoint struct {
	value          float64
	time           time.Time
	timeCoordinate int
}

type timeLine struct {
	points []timePoint
	color  ui.Color
	label  string
}

type timeRange struct {
	max time.Time
	min time.Time
}

type valueExtrema struct {
	max float64
	min float64
}

func NewRunChart(title string, precision int, refreshRateMs int) *RunChart {
	block := *ui.NewBlock()
	block.Title = title
	return &RunChart{
		Block:     block,
		lines:     []timeLine{},
		mutex:     &sync.Mutex{},
		precision: precision,
		timescale: calculateTimescale(refreshRateMs),
	}
}

func (self *RunChart) newChartGrid() chartGrid {

	linesCount := (self.Inner.Max.X - self.Inner.Min.X - self.grid.minTimeWidth) / xAxisGridWidth
	timeRange := getTimeRange(linesCount, self.timescale)

	return chartGrid{
		timeRange:    timeRange,
		valueExtrema: getValueExtrema(self.lines, timeRange),
		linesCount:   linesCount,
		paddingWidth: xAxisGridWidth,
		maxTimeWidth: self.Inner.Max.X,
		minTimeWidth: self.getMaxValueLength(),
	}
}

func (self *RunChart) newTimePoint(value float64) timePoint {
	now := time.Now()
	return timePoint{
		value:          value,
		time:           now,
		timeCoordinate: self.calculateTimeCoordinate(now),
	}
}

func (self *RunChart) Draw(buffer *ui.Buffer) {

	self.mutex.Lock()
	self.Block.Draw(buffer)
	self.grid = self.newChartGrid()

	drawArea := image.Rect(
		self.Inner.Min.X+self.grid.minTimeWidth+1, self.Inner.Min.Y,
		self.Inner.Max.X, self.Inner.Max.Y-xAxisLabelsHeight-1,
	)

	self.renderAxes(buffer)
	self.renderLines(buffer, drawArea)
	self.renderLegend(buffer, drawArea)
	self.mutex.Unlock()
}

func (self *RunChart) ConsumeSample(sample data.Sample) {

	float, err := strconv.ParseFloat(sample.Value, 64)

	if err != nil {
		// TODO visual notification + check sample.Error
	}

	self.mutex.Lock()

	lineIndex := -1

	for i, line := range self.lines {
		if line.label == sample.Label {
			lineIndex = i
		}
	}

	if lineIndex == -1 {
		line := &timeLine{
			points: []timePoint{},
			color:  sample.Color,
			label:  sample.Label,
		}
		self.lines = append(self.lines, *line)
		lineIndex = len(self.lines) - 1
	}

	line := self.lines[lineIndex]
	timePoint := self.newTimePoint(float)
	line.points = append(line.points, timePoint)
	self.lines[lineIndex] = line

	self.trimOutOfRangeValues()
	self.mutex.Unlock()
}

func (self *RunChart) renderLines(buffer *ui.Buffer, drawArea image.Rectangle) {

	canvas := ui.NewCanvas()
	canvas.Rectangle = drawArea

	if len(self.lines) == 0 || len(self.lines[0].points) == 0 {
		return
	}

	probe := self.lines[0].points[0]
	delta := ui.AbsInt(self.calculateTimeCoordinate(probe.time) - probe.timeCoordinate)

	for _, line := range self.lines {

		xToPoint := make(map[int]image.Point)
		pointsOrder := make([]int, 0)

		for i, timePoint := range line.points {

			timePoint.timeCoordinate = timePoint.timeCoordinate - delta
			line.points[i] = timePoint

			var y int
			if self.grid.valueExtrema.max == self.grid.valueExtrema.min {
				y = (drawArea.Dy() - 2) / 2
			} else {
				valuePerY := (self.grid.valueExtrema.max - self.grid.valueExtrema.min) / float64(drawArea.Dy()-2)
				y = int(float64(timePoint.value-self.grid.valueExtrema.min) / valuePerY)
			}

			point := image.Pt(timePoint.timeCoordinate, drawArea.Max.Y-y-1)

			if _, exists := xToPoint[point.X]; exists {
				continue
			}

			if !point.In(drawArea) {
				continue
			}

			xToPoint[point.X] = point
			pointsOrder = append(pointsOrder, point.X)
		}

		for i, x := range pointsOrder {

			currentPoint := xToPoint[x]
			var previousPoint image.Point

			if i == 0 {
				previousPoint = currentPoint
			} else {
				previousPoint = xToPoint[pointsOrder[i-1]]
			}

			canvas.Line(
				braillePoint(previousPoint),
				braillePoint(currentPoint),
				line.color,
			)
		}
	}

	canvas.Draw(buffer)
}

func (self *RunChart) renderAxes(buffer *ui.Buffer) {
	// draw origin cell
	buffer.SetCell(
		ui.NewCell(ui.BOTTOM_LEFT, ui.NewStyle(ui.ColorWhite)),
		image.Pt(self.Inner.Min.X+self.grid.minTimeWidth, self.Inner.Max.Y-xAxisLabelsHeight-1),
	)

	// draw x axis line
	for i := self.grid.minTimeWidth + 1; i < self.Inner.Dx(); i++ {
		buffer.SetCell(
			ui.NewCell(ui.HORIZONTAL_DASH, ui.NewStyle(ui.ColorWhite)),
			image.Pt(i+self.Inner.Min.X, self.Inner.Max.Y-xAxisLabelsHeight-1),
		)
	}

	// draw grid lines
	for y := 0; y < self.Inner.Dy()-xAxisLabelsHeight-2; y = y + 2 {
		for x := 1; x <= self.grid.linesCount; x++ {
			buffer.SetCell(
				ui.NewCell(ui.VERTICAL_DASH, ui.NewStyle(console.ColorDarkGrey)),
				image.Pt(self.grid.maxTimeWidth-x*xAxisGridWidth, y+self.Inner.Min.Y+1),
			)
		}
	}

	// draw y axis line
	for i := 0; i < self.Inner.Dy()-xAxisLabelsHeight-1; i++ {
		buffer.SetCell(
			ui.NewCell(ui.VERTICAL_DASH, ui.NewStyle(ui.ColorWhite)),
			image.Pt(self.Inner.Min.X+self.grid.minTimeWidth, i+self.Inner.Min.Y),
		)
	}

	// draw x axis time labels
	for i := 1; i <= self.grid.linesCount; i++ {
		labelTime := self.grid.timeRange.max.Add(time.Duration(-i) * self.timescale)
		buffer.SetString(
			labelTime.Format("15:04:05"),
			ui.NewStyle(ui.ColorWhite),
			image.Pt(self.grid.maxTimeWidth-xAxisLabelsWidth/2-i*(xAxisGridWidth), self.Inner.Max.Y-1),
		)
	}

	// draw y axis labels
	if self.grid.valueExtrema.max != self.grid.valueExtrema.min {
		labelsCount := (self.Inner.Dy() - xAxisLabelsHeight - 1) / (yAxisLabelsGap + yAxisLabelsHeight)
		valuePerY := (self.grid.valueExtrema.max - self.grid.valueExtrema.min) / float64(self.Inner.Dy()-xAxisLabelsHeight-3)
		for i := 0; i < int(labelsCount); i++ {
			value := self.grid.valueExtrema.max - (valuePerY * float64(i) * (yAxisLabelsGap + yAxisLabelsHeight))
			buffer.SetString(
				formatValue(value, self.precision),
				ui.NewStyle(ui.ColorWhite),
				image.Pt(self.Inner.Min.X, 1+self.Inner.Min.Y+i*(yAxisLabelsGap+yAxisLabelsHeight)),
			)
		}
	} else {
		buffer.SetString(
			formatValue(self.grid.valueExtrema.max, self.precision),
			ui.NewStyle(ui.ColorWhite),
			image.Pt(self.Inner.Min.X, self.Inner.Dy()/2))
	}
}

func (self *RunChart) renderLegend(buffer *ui.Buffer, rectangle image.Rectangle) {

	for i, line := range self.lines {

		extremum := getLineValueExtremum(line.points)

		buffer.SetString(
			string(ui.DOT),
			ui.NewStyle(line.color),
			image.Pt(self.Inner.Max.X-xAxisLegendWidth-2, self.Inner.Min.Y+1+i*5),
		)
		buffer.SetString(
			fmt.Sprintf("%s", line.label),
			ui.NewStyle(line.color),
			image.Pt(self.Inner.Max.X-xAxisLegendWidth, self.Inner.Min.Y+1+i*5),
		)
		buffer.SetString(
			fmt.Sprintf("cur %s", formatValue(line.points[len(line.points)-1].value, self.precision)),
			ui.NewStyle(ui.ColorWhite),
			image.Pt(self.Inner.Max.X-xAxisLegendWidth, self.Inner.Min.Y+2+i*5),
		)
		buffer.SetString(
			fmt.Sprintf("max %s", formatValue(extremum.max, self.precision)),
			ui.NewStyle(ui.ColorWhite),
			image.Pt(self.Inner.Max.X-xAxisLegendWidth, self.Inner.Min.Y+3+i*5),
		)
		buffer.SetString(
			fmt.Sprintf("min %s", formatValue(extremum.min, self.precision)),
			ui.NewStyle(ui.ColorWhite),
			image.Pt(self.Inner.Max.X-xAxisLegendWidth, self.Inner.Min.Y+4+i*5),
		)
	}
}

func (self *RunChart) trimOutOfRangeValues() {
	// TODO use hard limit
}

func (self *RunChart) calculateTimeCoordinate(t time.Time) int {
	timeDeltaWithGridMaxTime := self.grid.timeRange.max.Sub(t).Nanoseconds()
	timeDeltaToPaddingRelation := float64(timeDeltaWithGridMaxTime) / float64(self.timescale.Nanoseconds())
	return self.grid.maxTimeWidth - (int(float64(xAxisGridWidth) * timeDeltaToPaddingRelation))
}

func (self *RunChart) getMaxValueLength() int {

	maxValueLength := 0

	for _, line := range self.lines {
		for _, point := range line.points {
			l := len(formatValue(point.value, self.precision))
			if l > maxValueLength {
				maxValueLength = l
			}
		}
	}

	return maxValueLength
}

func formatValue(value float64, precision int) string {
	format := "%." + strconv.Itoa(precision) + "f"
	return fmt.Sprintf(format, value)
}

func getValueExtrema(items []timeLine, timeRange timeRange) valueExtrema {

	if len(items) == 0 {
		return valueExtrema{0, 0}
	}

	var max, min = -math.MaxFloat64, math.MaxFloat64

	for _, item := range items {
		for _, point := range item.points {
			if point.value > max && timeRange.isInRange(point.time) {
				max = point.value
			}
			if point.value < min && timeRange.isInRange(point.time) {
				min = point.value
			}
		}
	}

	return valueExtrema{max: max, min: min}
}

func (r *timeRange) isInRange(time time.Time) bool {
	return time.After(r.min) && time.Before(r.max)
}

func getLineValueExtremum(points []timePoint) valueExtrema {

	if len(points) == 0 {
		return valueExtrema{0, 0}
	}

	var max, min = -math.MaxFloat64, math.MaxFloat64

	for _, point := range points {
		if point.value > max {
			max = point.value
		}
		if point.value < min {
			min = point.value
		}
	}

	return valueExtrema{max: max, min: min}
}

func getTimeRange(linesCount int, scale time.Duration) timeRange {
	maxTime := time.Now()
	return timeRange{
		max: maxTime,
		min: maxTime.Add(-time.Duration(scale.Nanoseconds() * int64(linesCount))),
	}
}

func calculateTimescale(refreshRateMs int) time.Duration {

	multiplier := refreshRateMs * xAxisGridWidth / 2
	timescale := time.Duration(time.Millisecond * time.Duration(multiplier)).Round(time.Second)

	if timescale.Seconds() == 0 {
		return time.Second
	} else {
		return timescale
	}
}
