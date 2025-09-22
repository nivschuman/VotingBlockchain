package app

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/nodes"
	"github.com/nivschuman/VotingBlockchain/internal/ui/tabs"
	"github.com/nivschuman/VotingBlockchain/internal/voters"
)

type AppBuilder interface {
	BuildApp() App
}

type AppBuilderImpl struct {
	blockRepository       repositories.BlockRepository
	transactionRepository repositories.TransactionRepository
	voters                []*voters.Voter
	node                  nodes.Node
	config                *config.Config
}

type App interface {
	Start()
}

type AppImpl struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
}

func NewAppBuilderImpl(config *config.Config, node nodes.Node) *AppBuilderImpl {
	vtrs, err := voters.VotersFromJSONFile(config.VotersConfig.File)
	if err != nil {
		log.Printf("|App Builder| Failed to load voters: %v", err)
		vtrs = make([]*voters.Voter, 0)
	}

	return &AppBuilderImpl{
		transactionRepository: node.GetTransactionRepository(),
		blockRepository:       node.GetBlockRepository(),
		voters:                vtrs,
		node:                  node,
		config:                config,
	}
}

func (appBuilder *AppBuilderImpl) BuildApp() App {
	a := app.New()
	w := a.NewWindow("Blockchain")

	t := container.NewAppTabs()

	blocksTab := tabs.NewBlocksTab(appBuilder.blockRepository)
	t.Append(container.NewTabItem("Blocks", blocksTab.GetWidget()))

	transactionsTab := tabs.NewTransactionsTab(appBuilder.node, appBuilder.voters)
	t.Append(container.NewTabItem("Transactions", transactionsTab.GetWidget()))

	mempoolTab := tabs.NewMempoolTab(appBuilder.node)
	t.Append(container.NewTabItem("Mempool", mempoolTab.GetWidget()))

	peersTab := tabs.NewPeersTab(appBuilder.node.GetNetwork())
	t.Append(container.NewTabItem("Peers", peersTab.GetWidget()))

	addressesTab := tabs.NewAddressesTab(appBuilder.node.GetNetwork().GetAddressRepository())
	t.Append(container.NewTabItem("Addresses", addressesTab.GetWidget()))

	if appBuilder.config.MinerConfig.Enabled {
		miningTab := tabs.NewMiningTab(appBuilder.node.GetMiner())
		t.Append(container.NewTabItem("Mining", miningTab.GetWidget()))
	}

	votesTab := tabs.NewVotesTab(appBuilder.node)
	t.Append(container.NewTabItem("Votes", votesTab.GetWidget()))

	w.SetContent(t)
	w.Resize(fyne.NewSize(800, 600))

	icon, err := fyne.LoadResourceFromPath("assets/icon.png")
	if err == nil {
		w.SetIcon(icon)
	}

	return &AppImpl{fyneApp: a, mainWindow: w}
}

func (app *AppImpl) Start() {
	app.mainWindow.ShowAndRun()
}
