package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Represents the user's current action.
type State int16

const (
	READY  State = iota // User can start typing
	TYPING              // User is typing
	DONE                // Test completed
)

// Represents the contents being displayed to the user.
type View int16

const (
	PROMPT View = iota // Typing test
	STATS              // Calculated statistics
)

type tickMsg time.Time

// Styles
var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#999999"))
	mistakeStyle = lipgloss.NewStyle().Background(lipgloss.Color("#FF0000"))
	cursorStyle  = lipgloss.NewStyle().Background(lipgloss.Color("#e2b714")).Foreground(lipgloss.Color("#000000"))
)

// Default settings
var (
	terminalWidthDefault = 70
	timeLimitDefault     = 30
)

// Represents the application's state.
type Model struct {
	prompt     string // Randomly generated prompt
	userInput  string // The characters that the user has typed
	cursor     int    // User's position in the prompt
	mistakes   int    // Counter for typos
	charsTyped int    // Counter for characters typed
	timePassed int    // Counter for seconds passed
	timeLimit  int    // Time limit in seconds.
	view       View   // Current display
	state      State  // Current action
}

// The main entry point to the program.
func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("an error occurred: %v", err)
		os.Exit(1)
	}
}

func initialModel() Model {
	words, err := getWords("words/english.json")
	if err != nil {
		log.Fatalf("failed to get words: %v", err)
	}

	rand.Shuffle(len(words), func(i int, j int) {
		words[i], words[j] = words[j], words[i]
	})

	selection := words[:50]
	prompt := strings.Join(selection, " ")

	return Model{
		prompt:     prompt,
		userInput:  "",
		cursor:     0,
		mistakes:   0,
		charsTyped: 0,
		timePassed: 0,
		timeLimit:  timeLimitDefault,
		view:       PROMPT,
		state:      READY,
	}
}

// Get the words that will be used to construct the prompt.
func getWords(name string) ([]string, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var words []string
	if err := json.Unmarshal(data, &words); err != nil {
		return nil, fmt.Errorf("failed to parse json: %v", err)
	}

	return words, nil
}

// Runs once at the start of the application.
func (m Model) Init() tea.Cmd {
	return tick() // Starts the internal clock.
}

// Ticks are used to represent time throughout the program.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Manages the state of the application.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.view == STATS {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tickMsg:
		if m.state == TYPING {
			if m.timePassed >= m.timeLimit {
				m.state = DONE
				m.view = STATS
			}

			m.timePassed++

		}

		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "backspace":
			if m.state == TYPING {
				if m.cursor < 1 {
					return m, nil
				}

				m.cursor--
				m.userInput = m.userInput[:m.cursor]
			}

		default:
			r := msg.Runes

			if len(r) < 1 {
				return m, nil
			}

			switch m.state {
			case READY:
				m.state = TYPING
				fallthrough
			case TYPING:
				for i, c := range r {
					if c != []rune(m.prompt)[m.cursor+i] {
						m.mistakes++
					}
				}

				m.userInput += string(r)
				m.cursor += len(r)
				m.charsTyped += len(r)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	timeRemaining := m.timeLimit - m.timePassed
	s := ""

	switch m.view {
	case PROMPT:
		s += fmt.Sprintf("%v\n\n", timeRemaining)
		var readyToSplit = false
		for i, c := range m.prompt {
			if i >= terminalWidthDefault && i%terminalWidthDefault == 0 {
				readyToSplit = true
			}

			userInput := []rune(m.userInput)

			if i < len(userInput) {
				if userInput[i] == c {
					s += string(c)
				} else {
					s += mistakeStyle.Render(string(c))
				}
			} else if i == m.cursor {
				s += cursorStyle.Render(string(c))
			} else {
				s += promptStyle.Render(string(c))
			}

			if readyToSplit && c == ' ' {
				s += "\n"
				readyToSplit = false
			}
		}

		s += "\n\nPress ESC to quit\n"
	case STATS:
		s += "\n"
		correct := m.charsTyped - m.mistakes
		correctWords := float32(correct) / 5.0
		wpm := correctWords * (60.0 / float32(m.timePassed))
		s += fmt.Sprintf("WPM: %.2f\n", wpm)

		correctWords = float32(m.charsTyped) / 5.0
		raw := correctWords * (60.0 / float32(m.timePassed))
		s += fmt.Sprintf("Raw: %.2f\n", raw)

		var accuracy float32

		if m.charsTyped < 1 {
			accuracy = 0
		} else {
			accuracy = (1.0 - (float32(m.mistakes) / float32(m.charsTyped))) * 100.0
		}

		s += fmt.Sprintf("Accuracy: %.2f%%", accuracy)
		s += fmt.Sprintf(
			" (Correct: %v | Incorrect: %v)\n",
			(m.charsTyped - m.mistakes),
			m.mistakes,
		)
		s += "\n"
	}

	return s
}
