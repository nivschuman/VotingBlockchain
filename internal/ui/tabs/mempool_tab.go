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
)

type MempoolTab struct {
	pageSize int
	node     nodes.Node

	widget fyne.CanvasObject

	mempoolTable *widget.Table
	mempoolTxs   []*models.Transaction
	mempoolPage  int

	prevBtn    *widget.Button
	nextBtn    *widget.Button
	refreshBtn *widget.Button
}

func NewMempoolTab(node nodes.Node) *MempoolTab {
	m := &MempoolTab{
		node:     node,
		pageSize: 10,
	}
	m.widget = m.buildUI()
	m.refreshMempoolTransactions()
	return m
}

func (m *MempoolTab) buildUI() fyne.CanvasObject {
	m.refreshBtn = widget.NewButton("Refresh", m.refreshMempoolTransactions)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Mempool Transactions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		m.refreshBtn,
	)

	m.mempoolTable = widget.NewTable(
		func() (int, int) { return len(m.mempoolTxs) + 1, 4 },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
			return lbl
		},
		m.updateMempoolCell,
	)
	m.mempoolTable.SetColumnWidth(0, 250)
	m.mempoolTable.SetColumnWidth(1, 250)
	m.mempoolTable.SetColumnWidth(2, 130)
	m.mempoolTable.SetColumnWidth(3, 70)

	m.prevBtn = widget.NewButton("Prev", func() {
		if m.mempoolPage > 0 {
			m.mempoolPage--
			m.refreshMempoolTransactions()
		}
	})
	m.nextBtn = widget.NewButton("Next", func() {
		m.mempoolPage++
		m.refreshMempoolTransactions()
	})

	nav := container.NewHBox(m.prevBtn, m.nextBtn)

	scroll := container.NewVScroll(m.mempoolTable)
	scroll.SetMinSize(fyne.NewSize(700, 200))

	content := container.NewVBox(
		header,
		scroll,
		nav,
	)

	return container.NewPadded(content)
}

func (m *MempoolTab) refreshMempoolTransactions() {
	offset := m.mempoolPage * m.pageSize
	txs, _, err := m.node.GetTransactionRepository().GetMempoolPaged(offset, m.pageSize)
	if err != nil {
		m.mempoolTxs = []*models.Transaction{}
	} else {
		m.mempoolTxs = txs
	}

	if m.mempoolPage > 0 {
		m.prevBtn.Enable()
	} else {
		m.prevBtn.Disable()
	}

	if len(m.mempoolTxs) < m.pageSize {
		m.nextBtn.Disable()
	} else {
		m.nextBtn.Enable()
	}

	m.mempoolTable.Refresh()
}

func (m *MempoolTab) updateMempoolCell(id widget.TableCellID, co fyne.CanvasObject) {
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
		tx := m.mempoolTxs[id.Row-1]
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

func (m *MempoolTab) GetWidget() fyne.CanvasObject {
	return m.widget
}
