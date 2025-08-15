package app

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/nivschuman/VotingBlockchain/internal/config"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/ui/tabs"
	"github.com/nivschuman/VotingBlockchain/internal/voters"
	"gorm.io/gorm"
)

type AppBuilder interface {
	BuildApp() App
}

type AppBuilderImpl struct {
	blockRepository       repositories.BlockRepository
	transactionRepository repositories.TransactionRepository
	voters                []*voters.Voter
}

type App interface {
	Start()
}

type AppImpl struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
}

func NewAppBuilderImpl(db *gorm.DB, config *config.Config) *AppBuilderImpl {
	transactionRepository := repositories.NewTransactionRepositoryImpl(db)
	blockRepository := repositories.NewBlockRepositoryImpl(db, transactionRepository)

	vtrs, err := voters.VotersFromJSONFile(config.VotersConfig.File)
	if err != nil {
		log.Printf("|App Builder| Failed to load voters: %v", err)
		vtrs = make([]*voters.Voter, 0)
	}

	return &AppBuilderImpl{
		transactionRepository: transactionRepository,
		blockRepository:       blockRepository,
		voters:                vtrs,
	}
}

func (appBuilder *AppBuilderImpl) BuildApp() App {
	a := app.New()
	w := a.NewWindow("Blockchain UI")

	t := container.NewAppTabs()

	blocksTab := tabs.NewBlocksTab(appBuilder.blockRepository)
	t.Append(container.NewTabItem("Blocks", blocksTab.GetWidget()))

	transactionsTab := tabs.NewTransactionsTab(appBuilder.transactionRepository, appBuilder.voters)
	t.Append(container.NewTabItem("Transactions", transactionsTab.GetWidget()))

	w.SetContent(t)
	w.Resize(fyne.NewSize(800, 600))
	return &AppImpl{fyneApp: a, mainWindow: w}
}

func (app *AppImpl) Start() {
	app.mainWindow.ShowAndRun()
}
