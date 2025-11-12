package tabs

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/nivschuman/VotingBlockchain/internal/models"
	"github.com/nivschuman/VotingBlockchain/internal/nodes"
	"github.com/nivschuman/VotingBlockchain/internal/voters"
)

type TransactionsTab struct {
	pageSize int
	node     nodes.Node

	widget fyne.CanvasObject

	candidateIDEntry *widget.Entry
	voterSelect      *widget.Select
	generateBtn      *widget.Button
	refreshBtn       *widget.Button

	confirmedTable                     *widget.Table
	confirmedTxs                       []*models.Transaction
	confirmedPage                      int
	confirmedPrevBtn, confirmedNextBtn *widget.Button

	voters        []*voters.Voter
	selectedVoter *voters.Voter
}

func NewTransactionsTab(node nodes.Node, voters []*voters.Voter) *TransactionsTab {
	t := &TransactionsTab{
		node:     node,
		pageSize: 10,
		voters:   voters,
	}
	t.widget = t.buildUI()
	t.refreshConfirmedTransactions()
	return t
}

func (t *TransactionsTab) buildUI() fyne.CanvasObject {
	// Candidate ID entry
	t.candidateIDEntry = widget.NewEntry()
	t.candidateIDEntry.SetPlaceHolder("Candidate ID")
	t.candidateIDEntry.Resize(fyne.NewSize(120, 36))
	t.candidateIDEntry.Move(fyne.NewPos(210, 0))

	// Voter select
	voterNames := make([]string, len(t.voters))
	for i, v := range t.voters {
		voterNames[i] = v.Name
	}
	t.voterSelect = widget.NewSelect(voterNames, func(name string) {
		for _, v := range t.voters {
			if v.Name == name {
				t.selectedVoter = v
				break
			}
		}
	})
	if len(t.voters) > 0 {
		t.selectedVoter = t.voters[0]
		t.voterSelect.SetSelected(t.voters[0].Name)
	}
	t.voterSelect.Resize(fyne.NewSize(200, 36))
	t.voterSelect.Move(fyne.NewPos(0, 0))

	t.generateBtn = widget.NewButton("Generate Transaction", func() {
		if err := t.generateTransaction(); err != nil {
			fmt.Println("Failed to generate transaction:", err)
		}
		t.refreshConfirmedTransactions()
	})
	t.generateBtn.Resize(fyne.NewSize(170, 36))
	t.generateBtn.Move(fyne.NewPos(350, 0))

	t.refreshBtn = widget.NewButton("Refresh", t.refreshConfirmedTransactions)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Transactions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		t.refreshBtn,
	)

	inputSection := container.NewWithoutLayout(
		t.voterSelect,
		t.candidateIDEntry,
		t.generateBtn,
	)

	t.confirmedTable = widget.NewTable(
		func() (int, int) { return len(t.confirmedTxs) + 1, 4 },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
			return lbl
		},
		t.updateConfirmedCell,
	)
	t.confirmedTable.SetColumnWidth(0, 200)
	t.confirmedTable.SetColumnWidth(1, 200)
	t.confirmedTable.SetColumnWidth(2, 150)
	t.confirmedTable.SetColumnWidth(3, 100)

	t.confirmedPrevBtn = widget.NewButton("Prev", func() {
		if t.confirmedPage > 0 {
			t.confirmedPage--
			t.refreshConfirmedTransactions()
		}
	})
	t.confirmedNextBtn = widget.NewButton("Next", func() {
		t.confirmedPage++
		t.refreshConfirmedTransactions()
	})
	confirmedNav := container.NewHBox(t.confirmedPrevBtn, t.confirmedNextBtn)

	confirmedScroll := container.NewVScroll(t.confirmedTable)
	confirmedScroll.SetMinSize(fyne.NewSize(700, 150))

	confirmedSection := container.NewVBox(
		widget.NewLabelWithStyle("Confirmed Transactions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		confirmedScroll,
		confirmedNav,
	)

	content := container.NewVBox(
		header,
		widget.NewLabel("Create Transaction"),
		inputSection,
		confirmedSection,
	)

	return container.NewPadded(content)
}

func (t *TransactionsTab) generateTransaction() error {
	if t.selectedVoter == nil {
		return fmt.Errorf("no voter selected")
	}
	if t.candidateIDEntry.Text == "" {
		return fmt.Errorf("no candidate id")
	}
	candidateID, err := strconv.ParseUint(t.candidateIDEntry.Text, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid candidate ID: %v", err)
	}
	tx := &models.Transaction{
		Version:             1,
		CandidateId:         uint32(candidateID),
		VoterPublicKey:      t.selectedVoter.KeyPair.PublicKey.AsBytes(),
		GovernmentSignature: t.selectedVoter.GovernmentSignature,
	}
	tx.SetId()
	sig, err := t.selectedVoter.KeyPair.PrivateKey.CreateSignature(tx.Id)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}
	tx.Signature = sig
	t.node.ProcessGeneratedTransaction(tx)
	return nil
}

func (t *TransactionsTab) refreshConfirmedTransactions() {
	offset := t.confirmedPage * t.pageSize
	confirmed, _, err := t.node.GetTransactionRepository().GetConfirmedTransactionsPaged(offset, t.pageSize)
	if err != nil {
		t.confirmedTxs = []*models.Transaction{}
	} else {
		t.confirmedTxs = confirmed
	}

	if t.confirmedPage > 0 {
		t.confirmedPrevBtn.Enable()
	} else {
		t.confirmedPrevBtn.Disable()
	}

	if len(t.confirmedTxs) < t.pageSize {
		t.confirmedNextBtn.Disable()
	} else {
		t.confirmedNextBtn.Enable()
	}

	t.confirmedTable.Refresh()
}

func (t *TransactionsTab) updateConfirmedCell(id widget.TableCellID, co fyne.CanvasObject) {
	lbl := co.(*widget.Label)
	if id.Row == 0 {
		switch id.Col {
		case 0:
			lbl.SetText("Tx ID")
		case 1:
			lbl.SetText("Voter Key")
		case 2:
			lbl.SetText("Candidate ID")
		case 3:
			lbl.SetText("Version")
		}
		lbl.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		tx := t.confirmedTxs[id.Row-1]
		switch id.Col {
		case 0:
			lbl.SetText(fmt.Sprintf("%x", tx.Id))
		case 1:
			lbl.SetText(fmt.Sprintf("%x", tx.VoterPublicKey))
		case 2:
			lbl.SetText(strconv.Itoa(int(tx.CandidateId)))
		case 3:
			lbl.SetText(strconv.Itoa(int(tx.Version)))
		}
		lbl.TextStyle = fyne.TextStyle{}
	}
}

func (t *TransactionsTab) GetWidget() fyne.CanvasObject {
	return t.widget
}
