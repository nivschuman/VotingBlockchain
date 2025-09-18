package tabs

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/nivschuman/VotingBlockchain/internal/mining"
)

type MiningTab struct {
	miner mining.Miner

	widget fyne.CanvasObject

	totalBlocksLabel       *widget.Label
	currentHashesLabel     *widget.Label
	lastNonceLabel         *widget.Label
	lastBlockTimeLabel     *widget.Label
	currentHashRateLabel   *widget.Label
	difficultyLabel        *widget.Label
	nbitsLabel             *widget.Label
	currentBlockStartLabel *widget.Label
	currentDurationLabel   *widget.Label

	stopTicker chan bool
}

func NewMiningTab(miner mining.Miner) *MiningTab {
	t := &MiningTab{
		miner:      miner,
		stopTicker: make(chan bool),
	}

	t.widget = t.buildUI()
	t.startUpdating()

	return t
}

func (t *MiningTab) buildUI() fyne.CanvasObject {
	t.totalBlocksLabel = widget.NewLabel("0")
	t.currentHashesLabel = widget.NewLabel("0")
	t.lastNonceLabel = widget.NewLabel("0")
	t.lastBlockTimeLabel = widget.NewLabel("0s")
	t.currentHashRateLabel = widget.NewLabel("0 Hashes/sec")
	t.difficultyLabel = widget.NewLabel("0")
	t.nbitsLabel = widget.NewLabel("0")
	t.currentBlockStartLabel = widget.NewLabel("N/A")
	t.currentDurationLabel = widget.NewLabel("0s")

	grid := container.NewGridWithColumns(2,
		widget.NewLabel("Total Blocks Mined:"), t.totalBlocksLabel,
		widget.NewLabel("Difficulty:"), t.difficultyLabel,
		widget.NewLabel("NBits:"), t.nbitsLabel,
		widget.NewLabel("Current Block Start:"), t.currentBlockStartLabel,
		widget.NewLabel("Current Mining Duration:"), t.currentDurationLabel,
		widget.NewLabel("Current Block Hashes Tried:"), t.currentHashesLabel,
		widget.NewLabel("Current Hash Rate:"), t.currentHashRateLabel,
		widget.NewLabel("Last Nonce:"), t.lastNonceLabel,
		widget.NewLabel("Last Block Time:"), t.lastBlockTimeLabel,
	)

	return container.NewVBox(
		widget.NewLabelWithStyle("Mining Statistics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		grid,
	)
}

func (t *MiningTab) startUpdating() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := t.miner.GetMiningStatistics()
				t.updateUI(stats)
			case <-t.stopTicker:
				return
			}
		}
	}()
}

func (t *MiningTab) updateUI(stats mining.MiningStatistics) {
	fyne.Do(func() {
		t.totalBlocksLabel.SetText(fmt.Sprintf("%d", stats.TotalBlocksMined))
		t.currentHashesLabel.SetText(fmt.Sprintf("%d", stats.CurrentBlockHashesTried))
		t.lastNonceLabel.SetText(fmt.Sprintf("%d", stats.LastNonce))
		t.lastBlockTimeLabel.SetText(stats.LastBlockTime().String())
		t.currentHashRateLabel.SetText(fmt.Sprintf("%.2f Hashes/sec", stats.CurrentHashRate()))
		t.difficultyLabel.SetText(fmt.Sprintf("%.2f", stats.Difficulty()))
		t.nbitsLabel.SetText(fmt.Sprintf("0x%08x", stats.CurrentNBits))

		if stats.CurrentBlockStart > 0 {
			startTime := time.Unix(0, stats.CurrentBlockStart)
			t.currentBlockStartLabel.SetText(startTime.Format("15:04:05"))
			duration := time.Since(startTime)
			t.currentDurationLabel.SetText(duration.String())
		} else {
			t.currentBlockStartLabel.SetText("N/A")
			t.currentDurationLabel.SetText("0s")
		}
	})
}

func (t *MiningTab) Stop() {
	close(t.stopTicker)
}

func (t *MiningTab) GetWidget() fyne.CanvasObject {
	return t.widget
}
