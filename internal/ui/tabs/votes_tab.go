package tabs

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/nivschuman/VotingBlockchain/internal/nodes"
	"github.com/nivschuman/VotingBlockchain/internal/voters"
)

type VotesTab struct {
	node nodes.Node

	widget     fyne.CanvasObject
	results    []*voters.VotingResult
	refreshBtn *widget.Button
	resultsBox *fyne.Container
}

func NewVotesTab(node nodes.Node) *VotesTab {
	t := &VotesTab{node: node}
	t.widget = t.buildUI()
	t.refreshResults()
	return t
}

func (t *VotesTab) buildUI() fyne.CanvasObject {
	t.resultsBox = container.NewVBox()
	t.refreshBtn = widget.NewButton("Refresh Results", func() {
		t.refreshResults()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Voting Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(t.refreshBtn),
		t.resultsBox,
	)

	return container.NewPadded(content)
}

func (t *VotesTab) refreshResults() {
	results, err := t.node.GetTransactionRepository().GetVotingResults()
	if err != nil {
		fmt.Println("Failed to load results:", err)
		t.results = []*voters.VotingResult{}
	} else {
		t.results = results
	}

	t.resultsBox.Objects = nil
	for _, r := range t.results {
		label := widget.NewLabel("Candidate " + strconv.Itoa(int(r.CandidateId)) + ": " + strconv.Itoa(r.Votes))
		t.resultsBox.Add(label)
	}
	t.resultsBox.Refresh()
}

func (t *VotesTab) GetWidget() fyne.CanvasObject {
	return t.widget
}
