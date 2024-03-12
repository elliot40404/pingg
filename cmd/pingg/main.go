package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Ping struct {
	Data     [][]float64
	CurrSize int
	MaxSize  int
	Avg      float64
	Max      float64
	Min      float64
}

func NewPing(maxSize int) *Ping {
	return &Ping{
		Data:     make([][]float64, 1),
		CurrSize: 0,
		MaxSize:  maxSize,
		Avg:      0,
		Max:      0,
		Min:      0,
	}
}

func (p *Ping) AddData(d float64) {
	if p.CurrSize > 2 {
		p.Avg = (p.Avg*float64(p.CurrSize) + d) / float64(p.CurrSize+1)
		p.Max = max(p.Max, d)
		if p.Min == 0 {
			p.Min = d
		} else {
			p.Min = min(p.Min, d)
		}
	}
	if p.CurrSize < p.MaxSize {
		p.Data[0] = append(p.Data[0], d)
		p.CurrSize++
	} else {
		p.Data[0] = append(p.Data[0][1:], d)
	}
}

func (p *Ping) renderGraph(inpChan <-chan float64, cancel context.CancelFunc) {
	if p.CurrSize == 0 {
		log.Fatal("No data to render")
	}
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	x, y := ui.TerminalDimensions()

	p1 := widgets.NewParagraph()
	p1.Title = "Statistics"
	p1.Text = fmt.Sprintf("Avg: %.2fms Max: %.2fms Min: %.2fms", p.Avg, p.Max, p.Min)
	p1.TextStyle.Fg = ui.ColorYellow
	p1.SetRect(0, 0, x, 3)

	p0 := widgets.NewPlot()
	p0.Title = "Ping"
	p0.SetRect(0, 3, x, y)
	p0.Data = p.Data
	p0.DataLabels = []string{"Latency", "ms"}
	p0.PaddingLeft = 5
	p0.Border = false
	p0.LineColors[0] = ui.ColorGreen

	ui.Render(p0, p1)
	uiEvents := ui.PollEvents()
	for {
		select {
		case inp := <-inpChan:
			p.AddData(inp)
			p0.Data = p.Data
			p1.Text = fmt.Sprintf("Avg: %.2fms Max: %.2fms Min: %.2fms", p.Avg, p.Max, p.Min)
			ui.Render(p0, p1)
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				cancel()
				ui.Clear()
				return
			}
		}
	}
}

func parseLatency(timeStr string) (float64, error) {
	re := regexp.MustCompile(`(\d+\.?\d+?)\s*?ms`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("No match found in the time string")
	}
	timeValueStr := matches[1]
	timeValue, err := strconv.ParseFloat(timeValueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Error converting string to float: %v", err)
	}
	return timeValue, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ", os.Args[0], " <target>")
	}
	target := os.Args[1]
	ctx, cancel := context.WithCancel(context.Background())
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-t", target)
	} else {
		cmd = exec.CommandContext(ctx, "ping", target)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	inpChan := make(chan float64)
	go func() {
		for scanner.Scan() {
			data := scanner.Text()
			fdata, err := parseLatency(data)
			if err != nil {
				continue
			}
			inpChan <- fdata
		}
	}()
	p := NewPing(100)
	p.AddData(30)
	p.AddData(30)
	p.renderGraph(inpChan, cancel)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
