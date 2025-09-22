package tabs

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
)

type AddressesTab struct {
	addressRepository repositories.AddressRepository

	addressOffset   int
	addressTotal    int64
	addressPageSize int

	allAddresses []*db_models.AddressDB

	widget fyne.CanvasObject

	btnPrev *widget.Button
	btnNext *widget.Button
	lblPage *widget.Label

	addressScroll *container.Scroll
	refreshBtn    *widget.Button
}

func NewAddressesTab(addressRepository repositories.AddressRepository) *AddressesTab {
	tab := &AddressesTab{
		addressRepository: addressRepository,
		addressPageSize:   10,
	}
	tab.widget = tab.buildUI()
	tab.loadAddresses()
	return tab
}

func (tab *AddressesTab) makeCell(label string, w, h float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(color.Gray{Y: 200})
	rect.SetMinSize(fyne.NewSize(w, h))
	lbl := widget.NewLabel(label)
	lbl.Alignment = fyne.TextAlignLeading
	lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateEllipsis)
	lbl.Resize(fyne.NewSize(w, h))
	return container.NewWithoutLayout(rect, lbl)
}

func (tab *AddressesTab) buildUI() fyne.CanvasObject {
	tab.refreshBtn = widget.NewButton("Refresh", func() { tab.loadAddresses() })

	header := container.NewHBox(
		widget.NewLabelWithStyle("Known Addresses", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		tab.refreshBtn,
	)

	tab.btnPrev = widget.NewButton("< Prev", func() {
		if tab.addressOffset >= tab.addressPageSize {
			tab.addressOffset -= tab.addressPageSize
			tab.loadAddresses()
		}
	})
	tab.btnNext = widget.NewButton("Next >", func() {
		if int64(tab.addressOffset+tab.addressPageSize) < tab.addressTotal {
			tab.addressOffset += tab.addressPageSize
			tab.loadAddresses()
		}
	})
	tab.lblPage = widget.NewLabel("Page 1")
	addressNav := container.NewHBox(tab.btnPrev, tab.lblPage, tab.btnNext)

	tab.addressScroll = container.NewVScroll(container.NewVBox())
	tab.addressScroll.SetMinSize(fyne.NewSize(0, 250))

	content := container.NewVBox(
		header,
		tab.addressScroll,
		addressNav,
	)

	return container.NewPadded(content)
}

func (tab *AddressesTab) loadAddresses() {
	addrs, total, err := tab.addressRepository.GetAddressesPaged(tab.addressOffset, tab.addressPageSize, nil)
	if err != nil {
		fmt.Println("Error fetching addresses:", err)
		tab.allAddresses = []*db_models.AddressDB{}
		tab.addressTotal = 0
	} else {
		tab.allAddresses = addrs
		tab.addressTotal = total
	}

	// Update pagination label
	pageNum := (tab.addressOffset / tab.addressPageSize) + 1
	totalPages := (tab.addressTotal + int64(tab.addressPageSize) - 1) / int64(tab.addressPageSize)
	tab.lblPage.SetText(fmt.Sprintf("Page %d of %d", pageNum, totalPages))

	if tab.addressOffset == 0 {
		tab.btnPrev.Disable()
	} else {
		tab.btnPrev.Enable()
	}
	if int64(tab.addressOffset+tab.addressPageSize) >= tab.addressTotal {
		tab.btnNext.Disable()
	} else {
		tab.btnNext.Enable()
	}

	// Build rows
	rows := container.NewVBox()
	widths := []float32{150, 90, 90, 120}
	h := float32(30)

	// Header row
	headerGrid := container.NewGridWithColumns(len(widths),
		tab.makeCell("IP", widths[0], h),
		tab.makeCell("Port", widths[1], h),
		tab.makeCell("Node Type", widths[2], h),
		tab.makeCell("Last Seen", widths[3], h),
	)
	rows.Add(headerGrid)

	// Data rows
	for _, a := range tab.allAddresses {
		lastSeen := "N/A"
		if a.LastSeen != nil {
			lastSeen = a.LastSeen.String()
		}

		row := container.NewGridWithColumns(len(widths),
			tab.makeCell(a.Ip, widths[0], h),
			tab.makeCell(fmt.Sprintf("%d", a.Port), widths[1], h),
			tab.makeCell(fmt.Sprintf("%d", a.NodeType), widths[2], h),
			tab.makeCell(lastSeen, widths[3], h),
		)
		rows.Add(row)
	}

	tab.addressScroll.Content = rows
	tab.addressScroll.Refresh()
}

func (tab *AddressesTab) GetWidget() fyne.CanvasObject {
	return tab.widget
}
