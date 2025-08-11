package tabs

import (
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
)

const pageSize = 6

type BlocksTab struct {
	widget fyne.CanvasObject

	currentPage int64
	totalCount  int64

	offset int

	allBlocks []*models.BlockDB

	table        *widget.Table
	searchEntry  *widget.Entry
	sortAsc      bool
	chainView    *fyne.Container
	chainViewBox *fyne.Container

	btnPrev *widget.Button
	btnNext *widget.Button
	lblPage *widget.Label

	btnLeft  *widget.Button
	btnRight *widget.Button

	searchText string
}

func NewBlocksTab() *BlocksTab {
	t := &BlocksTab{
		sortAsc: true,
	}

	t.widget = t.buildUI()
	t.loadPage()
	t.load3Blocks()
	return t
}

func (t *BlocksTab) buildUI() fyne.CanvasObject {
	t.searchEntry = widget.NewEntry()
	t.searchEntry.SetPlaceHolder("Search")
	t.searchEntry.OnChanged = func(s string) {
		t.searchText = s
		t.currentPage = 0
		t.loadPage()
		t.load3Blocks()
	}

	fixedSearch := container.NewWithoutLayout(t.searchEntry)
	t.searchEntry.Resize(fyne.NewSize(100, 38))
	fixedSearch.Resize(fyne.NewSize(100, 38))

	refreshBtn := widget.NewButton("Refresh", func() {
		t.loadPage()
		t.load3Blocks()
	})
	refreshBtn.Importance = widget.LowImportance

	// Combine search and refresh horizontally
	searchBar := container.NewHBox(
		refreshBtn,
		fixedSearch,
		layout.NewSpacer(),
	)

	t.table = widget.NewTable(
		func() (int, int) {
			return len(t.allBlocks) + 1, 3
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("")
			lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
			return container.NewVBox(lbl)
		},
		func(ti widget.TableCellID, co fyne.CanvasObject) {
			cont := co.(*fyne.Container)

			if ti.Row == 0 {
				switch ti.Col {
				case 0:
					btn := widget.NewButton("Height", func() {
						t.sortAsc = !t.sortAsc
						t.loadPage()
						t.load3Blocks()
					})
					btn.Importance = widget.LowImportance
					btn.Alignment = widget.ButtonAlignLeading

					cont.Objects = nil
					cont.Add(btn)
					cont.Refresh()
					return
				case 1, 2:
					var label *widget.Label
					if len(cont.Objects) == 0 {
						label = widget.NewLabel("")
						cont.Add(label)
					} else {
						var ok bool
						label, ok = cont.Objects[0].(*widget.Label)
						if !ok {
							cont.Objects = nil
							label = widget.NewLabel("")
							cont.Add(label)
						}
					}

					if ti.Col == 1 {
						label.SetText("Block ID")
					} else {
						label.SetText("Miner Public Key")
					}
					label.Alignment = fyne.TextAlignLeading
					label.TextStyle = fyne.TextStyle{Bold: true}
					cont.Refresh()
					return
				}
			}

			// Data rows: ensure label present
			var label *widget.Label
			if len(cont.Objects) == 0 {
				label = widget.NewLabel("")
				cont.Add(label)
			} else {
				var ok bool
				label, ok = cont.Objects[0].(*widget.Label)
				if !ok {
					cont.Objects = nil
					label = widget.NewLabel("")
					cont.Add(label)
				}
			}

			block := t.allBlocks[ti.Row-1]
			switch ti.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", block.Height))
			case 1:
				label.SetText(fmt.Sprintf("%x", block.BlockHeaderId))
			case 2:
				label.SetText(fmt.Sprintf("%x", block.BlockHeader.MinerPublicKey))
			}
			label.Alignment = fyne.TextAlignLeading
			label.TextStyle = fyne.TextStyle{}

			cont.Refresh()
		},
	)

	// Set column widths for nice spacing
	t.table.SetColumnWidth(0, 90)
	t.table.SetColumnWidth(1, 280)
	t.table.SetColumnWidth(2, 280)

	tableScroll := container.NewVScroll(t.table)
	tableScroll.SetMinSize(fyne.NewSize(660, 300))
	centeredTable := container.NewCenter(tableScroll)

	t.btnPrev = widget.NewButton("< Prev", func() {
		if t.currentPage > 0 {
			t.currentPage--
			t.loadPage()
			t.load3Blocks()
		}
	})
	t.btnNext = widget.NewButton("Next >", func() {
		maxPage := (t.totalCount - 1) / pageSize
		if t.currentPage < maxPage {
			t.currentPage++
			t.loadPage()
			t.load3Blocks()
		}
	})
	t.lblPage = widget.NewLabel("Page 1")

	paginationControls := container.NewHBox(
		t.btnPrev,
		t.lblPage,
		t.btnNext,
		layout.NewSpacer(),
	)
	paginationControlsPadded := container.NewPadded(container.NewCenter(paginationControls))

	header := container.NewHBox(
		widget.NewLabelWithStyle("Blocks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)
	headerPadded := container.NewPadded(header)

	t.chainView = container.NewHBox()
	t.chainViewBox = container.NewVBox(t.chainView)
	chainViewPadded := container.NewPadded(t.chainViewBox)

	t.btnLeft = widget.NewButton("<", func() {
		t.offset++
		t.load3Blocks()
	})

	t.btnRight = widget.NewButton(">", func() {
		if t.offset > 0 {
			t.offset--
			t.load3Blocks()
		}
	})
	chainViewContainer := container.NewBorder(nil, nil, t.btnLeft, t.btnRight, chainViewPadded)
	centeredChainView := container.NewCenter(chainViewContainer)

	content := container.NewVBox(
		headerPadded,
		searchBar,
		centeredChainView,
		centeredTable,
		paginationControlsPadded,
	)

	return container.NewStack(
		container.NewPadded(content),
	)
}

func (t *BlocksTab) loadPage() {
	offset := int(t.currentPage * pageSize)
	blocks, totalCount, err := repositories.GlobalBlockRepository.GetActiveBlocksPaged(t.searchText, offset, pageSize, t.sortAsc)
	if err != nil {
		t.allBlocks = []*models.BlockDB{}
		t.totalCount = 0
	} else {
		t.allBlocks = blocks
		t.totalCount = totalCount
	}

	t.lblPage.SetText(fmt.Sprintf("Page %d of %d", t.currentPage+1, (t.totalCount+pageSize-1)/pageSize))
	t.updatePaginationButtons()
	t.table.Refresh()

	t.offset = 0
	t.updateChainButtons(totalCount)
}

func (t *BlocksTab) updatePaginationButtons() {
	if t.currentPage == 0 {
		t.btnPrev.Disable()
	} else {
		t.btnPrev.Enable()
	}

	maxPage := (t.totalCount - 1) / pageSize
	if t.currentPage >= maxPage {
		t.btnNext.Disable()
	} else {
		t.btnNext.Enable()
	}
}

func (t *BlocksTab) load3Blocks() {
	t.chainView.Objects = nil
	blocks, totalCount, err := repositories.GlobalBlockRepository.GetActiveBlocksPaged("", t.offset, 3, false)
	if err != nil || len(blocks) == 0 {
		t.chainView.Refresh()
		t.updateChainButtons(0)
		return
	}

	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	for i, block := range blocks {
		t.chainView.Add(t.makeBlockBox(block))
		if i < len(blocks)-1 {
			arrow := widget.NewLabel("←")
			arrow.Alignment = fyne.TextAlignCenter
			paddedArrow := container.NewVBox(layout.NewSpacer(), arrow, layout.NewSpacer())
			t.chainView.Add(paddedArrow)
		}
	}

	t.chainView.Refresh()
	t.updateChainButtons(totalCount)
}

func (t *BlocksTab) updateChainButtons(totalCount int64) {
	if t.offset >= int(totalCount)-3 {
		t.btnLeft.Disable()
	} else {
		t.btnLeft.Enable()
	}

	if t.offset <= 0 {
		t.btnRight.Disable()
	} else {
		t.btnRight.Enable()
	}
}

func (t *BlocksTab) makeBlockBox(block *models.BlockDB) fyne.CanvasObject {
	idShort := shortenBytes(block.BlockHeaderId, 10)
	height := block.Height

	textColor := color.NRGBA{R: 15, G: 32, B: 64, A: 255}

	heightText := canvas.NewText("Height: "+strconv.FormatUint(height, 10), textColor)
	heightText.TextStyle = fyne.TextStyle{Bold: true}
	heightText.Alignment = fyne.TextAlignCenter

	idText := canvas.NewText(idShort, textColor)
	idText.Alignment = fyne.TextAlignCenter

	boxContent := container.NewVBox(heightText, idText)

	background := canvas.NewRectangle(color.NRGBA{R: 220, G: 235, B: 255, A: 255})

	border := canvas.NewRectangle(color.NRGBA{R: 100, G: 120, B: 180, A: 255})
	border.StrokeColor = color.NRGBA{R: 50, G: 70, B: 140, A: 255}
	border.StrokeWidth = 2

	containerWithBorder := container.NewStack(background, border, container.NewPadded(boxContent))
	containerWithBorder.Resize(fyne.NewSize(150, 90))

	return containerWithBorder
}

func shortenBytes(data []byte, length int) string {
	const hexDigits = "0123456789abcdef"
	if len(data) == 0 {
		return ""
	}

	hex := make([]byte, len(data)*2)
	for i, b := range data {
		hex[i*2] = hexDigits[b>>4]
		hex[i*2+1] = hexDigits[b&0x0f]
	}

	str := string(hex)
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}

func (t *BlocksTab) GetWidget() fyne.CanvasObject {
	return t.widget
}
