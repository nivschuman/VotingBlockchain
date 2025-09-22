package tabs

import (
	"fmt"
	"image/color"
	"net"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	network "github.com/nivschuman/VotingBlockchain/internal/networking/network"
	peer "github.com/nivschuman/VotingBlockchain/internal/networking/peer"
)

type PeersTab struct {
	network network.Network

	allPeers []*peer.Peer

	widget fyne.CanvasObject

	refreshBtn *widget.Button

	ipEntry   *widget.Entry
	portEntry *widget.Entry
	dialBtn   *widget.Button

	peersScroll *container.Scroll
}

func NewPeersTab(network network.Network) *PeersTab {
	tab := &PeersTab{
		network: network,
	}
	tab.widget = tab.buildUI()
	tab.loadPeers()
	return tab
}

func (tab *PeersTab) makeCell(label string, w, h float32) fyne.CanvasObject {
	rect := canvas.NewRectangle(color.Gray{Y: 200})
	rect.SetMinSize(fyne.NewSize(w, h))
	lbl := widget.NewLabel(label)
	lbl.Alignment = fyne.TextAlignLeading
	lbl.Wrapping = fyne.TextWrap(fyne.TextTruncateEllipsis)
	lbl.Resize(fyne.NewSize(w, h))
	return container.NewWithoutLayout(rect, lbl)
}

func (tab *PeersTab) buildUI() fyne.CanvasObject {
	tab.refreshBtn = widget.NewButton("Refresh", func() { tab.loadPeers() })

	header := container.NewHBox(
		widget.NewLabelWithStyle("Connected Peers", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		tab.refreshBtn,
	)

	tab.ipEntry = widget.NewEntry()
	tab.ipEntry.SetPlaceHolder("IP")
	tab.ipEntry.Resize(fyne.NewSize(150, 36))
	tab.ipEntry.Move(fyne.NewPos(0, 0))

	tab.portEntry = widget.NewEntry()
	tab.portEntry.SetPlaceHolder("Port")
	tab.portEntry.Resize(fyne.NewSize(90, 36))
	tab.portEntry.Move(fyne.NewPos(160, 0))

	tab.dialBtn = widget.NewButton("Connect", func() {
		ip := net.ParseIP(tab.ipEntry.Text)
		port, err := strconv.Atoi(tab.portEntry.Text)
		if err != nil || ip == nil || port <= 0 {
			fmt.Println("Invalid IP or Port")
			return
		}
		addr := &models.Address{Ip: ip, Port: uint16(port)}
		if err := tab.network.DialAddress(addr); err != nil {
			fmt.Println("Dial failed:", err)
		} else {
			tab.loadPeers()
		}
	})
	tab.dialBtn.Resize(fyne.NewSize(100, 36))
	tab.dialBtn.Move(fyne.NewPos(260, 0))

	manualDial := container.NewWithoutLayout(tab.ipEntry, tab.portEntry, tab.dialBtn)

	tab.peersScroll = container.NewVScroll(container.NewVBox())
	tab.peersScroll.SetMinSize(fyne.NewSize(0, 200))

	content := container.NewVBox(
		header,
		tab.peersScroll,
		widget.NewLabel("Manual Connect"),
		manualDial,
	)

	return container.NewPadded(content)
}

func (tab *PeersTab) loadPeers() {
	tab.allPeers = tab.network.GetPeers()
	rows := container.NewVBox()
	widths := []float32{110, 90, 90, 90, 100, 90, 100}
	h := float32(30)

	headerGrid := container.NewGridWithColumns(len(widths),
		tab.makeCell("IP", widths[0], h),
		tab.makeCell("Port", widths[1], h),
		tab.makeCell("Node Type", widths[2], h),
		tab.makeCell("Version", widths[3], h),
		tab.makeCell("Time Offset", widths[4], h),
		tab.makeCell("Latency", widths[5], h),
		tab.makeCell("Action", widths[6], h),
	)
	rows.Add(headerGrid)

	for _, p := range tab.allPeers {
		btn := widget.NewButton("Remove", func(peerToRemove *peer.Peer) func() {
			return func() {
				tab.network.RemovePeer(peerToRemove)
				tab.loadPeers()
			}
		}(p))

		row := container.NewGridWithColumns(len(widths),
			tab.makeCell(p.Address.Ip.String(), widths[0], h),
			tab.makeCell(fmt.Sprintf("%d", p.Address.Port), widths[1], h),
			tab.makeCell(fmt.Sprintf("%d", p.Address.NodeType), widths[2], h),
			tab.makeCell(fmt.Sprintf("%d", p.PeerDetails.ProtocolVersion), widths[3], h),
			tab.makeCell(fmt.Sprintf("%ds", p.PeerDetails.TimeOffset), widths[4], h),
			tab.makeCell(fmt.Sprint(p.PingPongDetails.Latency.Round(time.Millisecond)), widths[5], h),
			btn,
		)
		rows.Add(row)
	}

	tab.peersScroll.Content = rows
	tab.peersScroll.Refresh()
}

func (tab *PeersTab) GetWidget() fyne.CanvasObject {
	return tab.widget
}
