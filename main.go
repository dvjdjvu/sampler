package main

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/djvu/sampler/asset"
	"github.com/djvu/sampler/client"
	"github.com/djvu/sampler/component"
	"github.com/djvu/sampler/component/asciibox"
	"github.com/djvu/sampler/component/barchart"
	"github.com/djvu/sampler/component/gauge"
	"github.com/djvu/sampler/component/layout"
	"github.com/djvu/sampler/component/runchart"
	"github.com/djvu/sampler/component/sparkline"
	"github.com/djvu/sampler/component/textbox"
	"github.com/djvu/sampler/config"
	"github.com/djvu/sampler/console"
	"github.com/djvu/sampler/data"
	"github.com/djvu/sampler/event"
	"github.com/djvu/sampler/metadata"
	"runtime/debug"
	"time"
)

type Starter struct {
	player  *asset.AudioPlayer
	lout    *layout.Layout
	palette console.Palette
	opt     config.Options
	cfg     config.Config
}

func (s *Starter) startAll() []*data.Sampler {
	samplers := make([]*data.Sampler, 0)
	for _, c := range s.cfg.RunCharts {
		cpt := runchart.NewRunChart(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, c.Items, c.Triggers))
	}
	for _, c := range s.cfg.SparkLines {
		cpt := sparkline.NewSparkLine(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, []config.Item{c.Item}, c.Triggers))
	}
	for _, c := range s.cfg.BarCharts {
		cpt := barchart.NewBarChart(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, c.Items, c.Triggers))
	}
	for _, c := range s.cfg.Gauges {
		cpt := gauge.NewGauge(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, []config.Item{c.Cur, c.Min, c.Max}, c.Triggers))
	}
	for _, c := range s.cfg.AsciiBoxes {
		cpt := asciibox.NewAsciiBox(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, []config.Item{c.Item}, c.Triggers))
	}
	for _, c := range s.cfg.TextBoxes {
		cpt := textbox.NewTextBox(c, s.palette)
		samplers = append(samplers, s.start(cpt, cpt.Consumer, c.ComponentConfig, []config.Item{c.Item}, c.Triggers))
	}
	return samplers
}

func (s *Starter) start(drawable ui.Drawable, consumer *data.Consumer, componentConfig config.ComponentConfig, itemsConfig []config.Item, triggersConfig []config.TriggerConfig) *data.Sampler {
	cpt := component.NewComponent(drawable, consumer, componentConfig)
	triggers := data.NewTriggers(triggersConfig, consumer, s.opt, s.player)
	items := data.NewItems(itemsConfig, *componentConfig.RateMs)
	s.lout.AddComponent(cpt)
	time.Sleep(10 * time.Millisecond) // desync coroutines
	return data.NewSampler(consumer, items, triggers, s.opt, s.cfg.Variables, *componentConfig.RateMs)
}

func main() {

	cfg, opt := config.LoadConfig()
	bc := client.NewBackendClient()

	statistics := metadata.GetStatistics(cfg)
	license := metadata.GetLicense()

	console.Init()
	defer console.Close()

	player := asset.NewAudioPlayer()
	if player != nil {
		defer player.Close()
	}

	defer handleCrash(statistics, opt, bc)
	defer updateStatistics(cfg, time.Now())

	palette := console.GetPalette(*cfg.Theme)
	lout := layout.NewLayout(component.NewStatusBar(*opt.ConfigFile, palette, license),
		component.NewMenu(palette), component.NewIntro(palette), component.NewNagWindow(palette))

	starter := &Starter{player, lout, palette, opt, *cfg}
	samplers := starter.startAll()

	handler := event.NewHandler(samplers, opt, lout)
	handler.HandleEvents()
}

func handleCrash(statistics *metadata.Statistics, opt config.Options, bc *client.BackendClient) {
	if rec := recover(); rec != nil {
		err := rec.(error)
		if !opt.DisableTelemetry {
			bc.ReportCrash(fmt.Sprintf("%s\n%s", err.Error(), string(debug.Stack())), statistics)
		}
		panic(err)
	}
}

func updateStatistics(cfg *config.Config, startTime time.Time) {
	metadata.PersistStatistics(cfg, time.Since(startTime))
}
