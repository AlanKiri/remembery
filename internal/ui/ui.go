package ui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alankiri/remembery/internal/beep"
	"github.com/alankiri/remembery/internal/config"
	"github.com/alankiri/remembery/internal/engine"
	"github.com/alankiri/remembery/internal/levels"
	"github.com/alankiri/remembery/internal/store"
	"github.com/alankiri/remembery/internal/ui/common"
	"github.com/alankiri/remembery/internal/ui/screen"
	"github.com/alankiri/remembery/internal/ui/views"
)

type Model struct {
	db     *store.DB
	eng    *engine.Engine
	cfg    config.Config
	levels []levels.Level
	beeper beep.Beep

	width, height int
	screen        screen.Screen
	errMsg        string

	welcome  views.WelcomeModel
	congrats views.CongratsModel
	list     views.ListModel

	new views.NewModel

	edit   views.EditModel
	delete views.DeleteModel

	early views.EarlyModel
	level views.LevelModel
	train views.TrainModel
	vault views.VaultModel
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	lvl := cfg.Levels
	for {
		db, err := store.New()
		if err != nil {
			return err
		}
		exists, err := db.VaultExists()
		if err != nil {
			return err
		}
		// If the user already dismissed the initial encryption prompt,
		// start without a vault and do not ask again.
		if !exists && cfg.PromptedForVault {
			m := New(db, cfg, lvl)
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err = p.Run()
			return err
		}
		uModel := newUnlockModel(db, !exists)
		pUnlock := tea.NewProgram(uModel, tea.WithAltScreen())
		final, err := pUnlock.Run()
		if err != nil {
			return err
		}
		um := final.(unlockModel)
		if um.reset {
			continue
		}
		if !um.skipped && um.result == nil {
			return errors.New("vault not unlocked")
		}
		if !um.skipped {
			db.SetVault(um.result)
		}
		cfg.PromptedForVault = true
		if err := config.Save(cfg); err != nil {
			return err
		}
		m := New(db, cfg, lvl)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	}
}

func New(db *store.DB, cfg config.Config, lvl []levels.Level) Model {
	m := Model{
		db:     db,
		eng:    engine.New(lvl),
		cfg:    cfg,
		levels: lvl,
		beeper: beep.Terminal{},
		screen: screen.ScreenWelcome,
	}

	m.list = views.NewListModel(m.db, m.eng, m.levels)
	if err := m.list.LoadTrainers(); err != nil {
		m.errMsg = err.Error()
	}
	due := 0
	for _, t := range m.list.Trainers {
		if m.list.IsDue(t) {
			due++
		}
	}
	m.welcome = views.NewWelcomeModel(due, len(m.list.Trainers), m.db.HasVault())
	return m
}

func (m Model) Init() tea.Cmd {
	return m.welcome.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - 6
		m.height = msg.Height - 1
		if m.width < 0 {
			m.width = 0
		}
		if m.height < 0 {
			m.height = 0
		}
		return m, nil
	case screen.ChangeScreenMsg:
		m.screen = msg.To
		m.errMsg = msg.Err
		if msg.To == screen.ScreenList {
			if err := m.list.LoadTrainers(); err != nil {
				m.errMsg = err.Error()
			}
		}
		if msg.To == screen.ScreenNew {
			m.new = views.NewNewModel(m.db, m.eng, m.levels)
		}
		if msg.To == screen.ScreenVault {
			m.vault = views.NewVaultModel(m.db, &m.cfg, m.eng, m.levels)
		}
		return m, nil
	case common.StartTrainMsg:
		train, err := views.NewTrainModel(m.db, m.eng, m.beeper, msg.Trainer, msg.Counts)
		if err != nil {
			return m, screen.ChangeScreen(screen.ScreenList, err.Error())
		}
		m.train = train
		m.screen = screen.ScreenTrain
		return m, nil
	case common.SetErrMsg:
		m.errMsg = msg.Text
		return m, nil
	case common.StartEditMsg:
		m.edit = views.NewEditModel(m.db, m.eng, m.levels, msg.Trainer)
		m.errMsg = ""
		m.screen = screen.ScreenEdit
		return m, nil
	case common.ShowCongratsMsg:
		m.congrats = views.NewCongratsModel(msg.Text, msg.NextScreen)
		if msg.LevelTrainer != nil {
			m.level = views.NewLevelModel(m.db, m.eng, msg.LevelTrainer, msg.LevelOffer)
		}
		m.screen = screen.ScreenCongrats
		m.errMsg = ""
		return m, nil
	case common.StartDeleteMsg:
		m.delete = views.NewDeleteModel(m.db, *msg.Trainer)
		m.screen = screen.ScreenDelete
		return m, nil
	case common.ShowEarlyMsg:
		m.early = views.NewEarlyModel(m.eng, msg.Trainer)
		m.screen = screen.ScreenEarlyTrain
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.screen == screen.ScreenWelcome {
			newWelcome, cmd := m.welcome.Update(msg)
			m.welcome = newWelcome
			return m, cmd
		}
		if m.screen == screen.ScreenList {
			newList, cmd := m.list.Update(msg)
			m.list = newList
			return m, cmd
		}
		if m.screen == screen.ScreenNew {
			newNew, cmd := m.new.Update(msg)
			m.new = newNew
			return m, cmd
		}
		if m.screen == screen.ScreenEdit {
			newEdit, cmd := m.edit.Update(msg)
			m.edit = newEdit
			return m, cmd
		}
		if m.screen == screen.ScreenDelete {
			newDelete, cmd := m.delete.Update(msg)
			m.delete = newDelete
			return m, cmd
		}
		if m.screen == screen.ScreenEarlyTrain {
			newEarly, cmd := m.early.Update(msg)
			m.early = newEarly
			return m, cmd
		}
		if m.screen == screen.ScreenTrain {
			newTrain, cmd := m.train.Update(msg)
			m.train = newTrain
			return m, cmd
		}
		if m.screen == screen.ScreenCongrats {
			newCongrats, cmd := m.congrats.Update(msg)
			m.congrats = newCongrats
			return m, cmd
		}
		if m.screen == screen.ScreenLevel {
			newLevel, cmd := m.level.Update(msg)
			m.level = newLevel
			return m, cmd
		}
		if m.screen == screen.ScreenVault {
			newVault, cmd := m.vault.Update(msg)
			m.vault = newVault
			return m, cmd
		}
	case common.WelcomeTickMsg:
		if m.screen != screen.ScreenWelcome {
			return m, nil
		}
		newWelcome, cmd := m.welcome.Update(msg)
		m.welcome = newWelcome
		return m, cmd
	case common.TickMsg:
		if m.screen == screen.ScreenTrain {
			newTrain, cmd := m.train.Update(msg)
			m.train = newTrain
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) View() string {
	var inner string
	switch m.screen {
	case screen.ScreenWelcome:
		inner = m.welcome.View(m.width, m.height)
	case screen.ScreenList:
		inner = m.list.View(m.width, m.height, m.errMsg)
	case screen.ScreenNew:
		inner = m.new.View(m.width, m.height, m.errMsg)
	case screen.ScreenEdit:
		inner = m.edit.View(m.width, m.height, m.errMsg)
	case screen.ScreenDelete:
		inner = m.delete.View(m.width, m.height)
	case screen.ScreenEarlyTrain:
		inner = m.early.View(m.width, m.height)
	case screen.ScreenTrain:
		inner = m.train.View(m.width, m.height)
	case screen.ScreenCongrats:
		inner = m.congrats.View()
	case screen.ScreenLevel:
		inner = m.level.View()
	case screen.ScreenVault:
		inner = m.vault.View(m.width, m.height)
	default:
		inner = "unknown screen"
	}
	return lipgloss.NewStyle().Padding(1, 3, 0, 3).Render(inner)
}
