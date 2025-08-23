package tabs

import (
	"fmt"
	"image/color"
	"net"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
)

type PeersTab struct {
	network network.Network

	addressOffset   int
	addressTotal    int64
	addressPageSize int

	allAddresses []*db_models.AddressDB
	allPeers     []*peer.Peer

	widget fyne.CanvasObject

	btnPrev *widget.Button
	btnNext *widget.Button
	lblPage *widget.Label

	peersScroll   *container.Scroll
	addressScroll *container.Scroll
	refreshBtn    *widget.Button
}

func NewPeersTab(network network.Network) *PeersTab {
	tab := &PeersTab{
		network:         network,
		addressPageSize: 10,
	}

	tab.widget = tab.buildUI()
	tab.loadAll()
	return tab
}

func borderCell(obj fyne.CanvasObject, w, h float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(color.Gray{Y: 200})
	rect.SetMinSize(fyne.NewSize(w, h))
	return container.NewWithoutLayout(rect, obj)
}

func fixedLabel(text string, w, h float32) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Alignment = fyne.TextAlignLeading
	lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	lbl.Resize(fyne.NewSize(w, h))
	return lbl
}

func (tab *PeersTab) buildUI() fyne.CanvasObject {
	tab.refreshBtn = widget.NewButton("Refresh All", func() { tab.loadAll() })
	tab.refreshBtn.Importance = widget.HighImportance
	header := container.NewHBox(
		widget.NewLabelWithStyle("Peers & Addresses", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		tab.refreshBtn,
	)

	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("IP")
	ipEntry.Resize(fyne.NewSize(150, 36))
	ipEntry.Move(fyne.NewPos(0, 0))

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Port")
	portEntry.Resize(fyne.NewSize(90, 36))
	portEntry.Move(fyne.NewPos(160, 0))

	dialBtn := widget.NewButton("Connect", func() {
		ip := net.ParseIP(ipEntry.Text)
		port, err := strconv.Atoi(portEntry.Text)
		if err != nil || ip == nil || port <= 0 {
			fmt.Println("Invalid IP or Port")
			return
		}
		addr := &models.Address{Ip: ip, Port: uint16(port)}
		if err := tab.network.DialAddress(addr); err != nil {
			fmt.Println("Dial failed:", err)
		} else {
			tab.loadAll()
		}
	})
	dialBtn.Resize(fyne.NewSize(100, 36))
	dialBtn.Move(fyne.NewPos(260, 0))
	manualDial := container.NewWithoutLayout(ipEntry, portEntry, dialBtn)

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

	addrIP := widget.NewEntry()
	addrIP.SetPlaceHolder("IP")
	addrIP.Resize(fyne.NewSize(150, 36))
	addrIP.Move(fyne.NewPos(0, 0))

	addrPort := widget.NewEntry()
	addrPort.SetPlaceHolder("Port")
	addrPort.Resize(fyne.NewSize(90, 36))
	addrPort.Move(fyne.NewPos(160, 0))

	addrNodeType := widget.NewEntry()
	addrNodeType.SetPlaceHolder("Node Type")
	addrNodeType.Resize(fyne.NewSize(100, 36))
	addrNodeType.Move(fyne.NewPos(260, 0))

	addAddrBtn := widget.NewButton("Add Address", func() {
		ip := net.ParseIP(addrIP.Text)
		port, err1 := strconv.Atoi(addrPort.Text)
		nodeType, err2 := strconv.Atoi(addrNodeType.Text)
		if err1 != nil || err2 != nil || ip == nil || port <= 0 || nodeType <= 0 {
			fmt.Println("Invalid input")
			return
		}
		addr := &models.Address{Ip: ip, Port: uint16(port), NodeType: uint32(nodeType)}
		if err := tab.network.GetAddressRepository().InsertIfNotExists(addr); err != nil {
			fmt.Println("Insert failed:", err)
		} else {
			tab.loadAll()
		}
	})
	addAddrBtn.Resize(fyne.NewSize(100, 36))
	addAddrBtn.Move(fyne.NewPos(370, 0))
	manualAddr := container.NewWithoutLayout(addrIP, addrPort, addrNodeType, addAddrBtn)

	tab.peersScroll = container.NewVScroll(container.NewVBox())
	tab.peersScroll.SetMinSize(fyne.NewSize(0, 150))
	tab.addressScroll = container.NewVScroll(container.NewVBox())
	tab.addressScroll.SetMinSize(fyne.NewSize(0, 150))

	content := container.NewVBox(
		header,
		widget.NewLabel("Connected Peers"),
		tab.peersScroll,
		widget.NewLabel("Manual Connect"),
		manualDial,
		widget.NewLabel("Known Addresses"),
		tab.addressScroll,
		addressNav,
		widget.NewLabel("Add Address Manually"),
		manualAddr,
	)

	return container.NewPadded(content)
}

func (tab *PeersTab) loadPeers() {
	tab.allPeers = tab.network.GetPeers()

	rows := container.NewVBox()

	widths := []float32{100, 90, 90, 90, 100, 90, 100}
	h := float32(30)

	headerGrid := container.NewGridWithColumns(len(widths),
		borderCell(fixedLabel("IP", widths[0], h), widths[0], h),
		borderCell(fixedLabel("Port", widths[1], h), widths[1], h),
		borderCell(fixedLabel("Node Type", widths[2], h), widths[2], h),
		borderCell(fixedLabel("Version", widths[3], h), widths[3], h),
		borderCell(fixedLabel("Time Offset", widths[4], h), widths[4], h),
		borderCell(fixedLabel("Latency", widths[5], h), widths[5], h),
		borderCell(fixedLabel("Action", widths[6], h), widths[6], h),
	)
	rows.Add(headerGrid)

	// Data rows
	for _, p := range tab.allPeers {
		btn := widget.NewButton("Remove", func(peerToRemove *peer.Peer) func() {
			return func() {
				tab.network.RemovePeer(peerToRemove)
				tab.loadAll()
			}
		}(p))

		row := container.NewGridWithColumns(len(widths),
			borderCell(fixedLabel(p.Address.Ip.String(), widths[0], h), widths[0], h),
			borderCell(fixedLabel(fmt.Sprintf("%d", p.Address.Port), widths[1], h), widths[1], h),
			borderCell(fixedLabel(fmt.Sprintf("%d", p.Address.NodeType), widths[2], h), widths[2], h),
			borderCell(fixedLabel(fmt.Sprintf("%d", p.PeerDetails.ProtocolVersion), widths[3], h), widths[3], h),
			borderCell(fixedLabel(fmt.Sprint(p.PeerDetails.TimeOffset), widths[4], h), widths[4], h),
			borderCell(fixedLabel(fmt.Sprint(p.PingPongDetails.Latency), widths[5], h), widths[5], h),
			borderCell(btn, widths[6], h),
		)
		rows.Add(row)
	}

	tab.peersScroll.Content = rows
	tab.peersScroll.Refresh()
}

func (tab *PeersTab) loadAddresses() {
	addrs, total, err := tab.network.GetAddressRepository().GetAddressesPaged(tab.addressOffset, tab.addressPageSize, nil)
	if err != nil {
		fmt.Println("Error fetching addresses:", err)
		tab.allAddresses = []*db_models.AddressDB{}
		tab.addressTotal = 0
	} else {
		tab.allAddresses = addrs
		tab.addressTotal = total
	}

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

	rows := container.NewVBox()
	widths := []float32{150, 90, 90, 120}
	h := float32(30)

	// Header
	headerGrid := container.NewGridWithColumns(len(widths),
		borderCell(fixedLabel("IP", widths[0], h), widths[0], h),
		borderCell(fixedLabel("Port", widths[1], h), widths[1], h),
		borderCell(fixedLabel("Node Type", widths[2], h), widths[2], h),
		borderCell(fixedLabel("Last Seen", widths[3], h), widths[3], h),
	)
	rows.Add(headerGrid)

	// Data rows
	for _, a := range tab.allAddresses {
		lastSeen := "N/A"
		if a.LastSeen != nil {
			lastSeen = a.LastSeen.String()
		}
		row := container.NewGridWithColumns(len(widths),
			borderCell(fixedLabel(a.Ip, widths[0], h), widths[0], h),
			borderCell(fixedLabel(fmt.Sprintf("%d", a.Port), widths[1], h), widths[1], h),
			borderCell(fixedLabel(fmt.Sprintf("%d", a.NodeType), widths[2], h), widths[2], h),
			borderCell(fixedLabel(lastSeen, widths[3], h), widths[3], h),
		)
		rows.Add(row)
	}

	tab.addressScroll.Content = rows
	tab.addressScroll.Refresh()
}

func (tab *PeersTab) loadAll() {
	tab.loadPeers()
	tab.loadAddresses()
}

func (tab *PeersTab) GetWidget() fyne.CanvasObject {
	return tab.widget
}
