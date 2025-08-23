package tabs

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	confirmedTable *widget.Table
	mempoolTable   *widget.Table

	confirmedTxs []*models.Transaction
	mempoolTxs   []*models.Transaction

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
	t.refreshTransactions()
	return t
}

func (t *TransactionsTab) buildUI() fyne.CanvasObject {
	// Candidate ID input
	t.candidateIDEntry = widget.NewEntry()
	t.candidateIDEntry.SetPlaceHolder("Candidate ID")

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

	// Buttons
	t.generateBtn = widget.NewButton("Generate Transaction", func() {
		if err := t.generateTransaction(); err != nil {
			fmt.Println("Failed to generate transaction:", err)
		}
		t.refreshTransactions()
	})
	t.refreshBtn = widget.NewButton("Refresh Transactions", func() {
		t.refreshTransactions()
	})

	inputSection := container.NewVBox(
		widget.NewLabel("Create Transaction"),
		t.voterSelect,
		t.candidateIDEntry,
		container.NewHBox(t.generateBtn, t.refreshBtn),
	)

	// Confirmed Transactions Table
	t.confirmedTable = widget.NewTable(
		func() (int, int) { return len(t.confirmedTxs) + 1, 4 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		t.updateConfirmedCell,
	)
	t.confirmedTable.SetColumnWidth(0, 200)
	t.confirmedTable.SetColumnWidth(1, 200)
	t.confirmedTable.SetColumnWidth(2, 150)
	t.confirmedTable.SetColumnWidth(3, 100)

	confirmedScroll := container.NewVScroll(t.confirmedTable)
	confirmedScroll.SetMinSize(fyne.NewSize(700, 100))

	// Mempool Transactions Table
	t.mempoolTable = widget.NewTable(
		func() (int, int) { return len(t.mempoolTxs) + 1, 4 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		t.updateMempoolCell,
	)
	t.mempoolTable.SetColumnWidth(0, 200)
	t.mempoolTable.SetColumnWidth(1, 200)
	t.mempoolTable.SetColumnWidth(2, 150)
	t.mempoolTable.SetColumnWidth(3, 100)

	mempoolScroll := container.NewVScroll(t.mempoolTable)
	mempoolScroll.SetMinSize(fyne.NewSize(700, 100))

	content := container.NewVBox(
		inputSection,
		widget.NewLabelWithStyle("Confirmed Transactions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		confirmedScroll,
		widget.NewLabelWithStyle("Mempool", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mempoolScroll,
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

	candidateIDStr := t.candidateIDEntry.Text
	candidateID, err := strconv.ParseUint(candidateIDStr, 10, 32)
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

func (t *TransactionsTab) refreshTransactions() {
	confirmed, _, err := t.node.GetTransactionRepository().GetConfirmedTransactionsPaged(0, t.pageSize)
	if err != nil {
		t.confirmedTxs = []*models.Transaction{}
	} else {
		t.confirmedTxs = confirmed
	}
	t.confirmedTable.Refresh()

	mempool, _, err := t.node.GetTransactionRepository().GetMempoolPaged(0, t.pageSize)
	if err != nil {
		t.mempoolTxs = []*models.Transaction{}
	} else {
		t.mempoolTxs = mempool
	}
	t.mempoolTable.Refresh()
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
			lbl.SetText(truncateHex(tx.Id, 16))
		case 1:
			lbl.SetText(truncateHex(tx.VoterPublicKey, 16))
		case 2:
			lbl.SetText(strconv.Itoa(int(tx.CandidateId)))
		case 3:
			lbl.SetText(strconv.Itoa(int(tx.Version)))
		}
		lbl.TextStyle = fyne.TextStyle{}
	}
}

func (t *TransactionsTab) updateMempoolCell(id widget.TableCellID, co fyne.CanvasObject) {
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
		tx := t.mempoolTxs[id.Row-1]
		switch id.Col {
		case 0:
			lbl.SetText(truncateHex(tx.Id, 16))
		case 1:
			lbl.SetText(truncateHex(tx.VoterPublicKey, 16))
		case 2:
			lbl.SetText(strconv.Itoa(int(tx.CandidateId)))
		case 3:
			lbl.SetText(strconv.Itoa(int(tx.Version)))
		}
		lbl.TextStyle = fyne.TextStyle{}
	}
}

func truncateHex(data []byte, length int) string {
	if len(data) == 0 {
		return ""
	}
	hexStr := fmt.Sprintf("%x", data)
	if len(hexStr) <= length {
		return hexStr
	}
	return hexStr[:length] + "..."
}

func (t *TransactionsTab) GetWidget() fyne.CanvasObject {
	return t.widget
}
