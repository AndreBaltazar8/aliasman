package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	aliasFileName = ".aliasman_aliases"
	tagStart      = "# START ALIASMAN MANAGED BLOCK"
	tagEnd        = "# END ALIASMAN MANAGED BLOCK"
)

func main() {
	app := tview.NewApplication()
	pages := tview.NewPages()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		os.Exit(1)
	}

	aliasFilePath := filepath.Join(homeDir, aliasFileName)
	shellConfigPath := detectShellConfig(homeDir)

	// Check installation and install if not already installed
	if !isAliasmanInstalled(aliasFilePath, shellConfigPath) {
		installAliasman(aliasFilePath, shellConfigPath)
	}

	mainMenu := createMainMenu(app, pages, aliasFilePath, shellConfigPath)
	pages.AddPage("main", mainMenu, true, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		fmt.Println("Error running application:", err)
		os.Exit(1)
	}
}

func createMainMenu(app *tview.Application, pages *tview.Pages, aliasFilePath, shellConfigPath string) *tview.List {
	mainMenu := tview.NewList().
		AddItem("Manage Aliases", "Add, remove, or list aliases", 'm', nil).
		AddItem("Check Installation", "Check if aliasman is installed", 'c', nil).
		AddItem("AI Assisted Alias Creation", "Create an alias using AI assistance", 'a', nil).
		AddItem("Quit", "Exit the application", 'q', func() {
			app.Stop()
			showReloadInstructions(shellConfigPath)
		})

	mainMenu.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		switch index {
		case 0:
			showAliasManagement(app, pages, aliasFilePath)
		case 1:
			checkInstallation(app, pages, aliasFilePath, shellConfigPath)
		case 2:
			showAIAssistedAliasCreation(app, pages, aliasFilePath)
		}
	})

	return mainMenu
}

func showReloadInstructions(shellConfigPath string) {
	reloadCommand := fmt.Sprintf("source %s", shellConfigPath)
	fmt.Printf("\nTo reload your aliases in the current shell, run this command:\n\n%s\n\n", reloadCommand)
}

func showAliasManagement(app *tview.Application, pages *tview.Pages, aliasFilePath string) {
	list := tview.NewList().
		AddItem("List Aliases", "Show all defined aliases", 'l', nil).
		AddItem("Add Alias", "Create a new alias", 'a', nil).
		AddItem("Back", "Return to main menu", 'q', nil)

	list.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		switch index {
		case 0:
			listAliases(app, pages, aliasFilePath)
		case 1:
			addAlias(app, pages, aliasFilePath)
		case 2:
			pages.SwitchToPage("main")
		}
	})

	pages.AddPage("aliasManagement", list, true, true)
	pages.SwitchToPage("aliasManagement")
}

func listAliases(app *tview.Application, pages *tview.Pages, aliasFilePath string) {
	aliases, err := readAliases(aliasFilePath)
	if err != nil {
		showErrorModal(app, pages, "Error reading aliases: "+err.Error())
		return
	}

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

	table.SetCell(0, 0, tview.NewTableCell("Alias").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	table.SetCell(0, 1, tview.NewTableCell("Command").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))

	for i, alias := range aliases {
		table.SetCell(i+1, 0, tview.NewTableCell(alias.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		table.SetCell(i+1, 1, tview.NewTableCell(alias.Command).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	}

	table.Select(1, 0).SetFixed(1, 0).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			pages.SwitchToPage("aliasManagement")
		}
	}).SetSelectedFunc(func(row, column int) {
		if row > 0 {
			aliasToDelete := aliases[row-1]
			deleteAlias(app, pages, aliasFilePath, aliasToDelete.Name)
		}
	})

	frame := tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0)
	frame.AddText("Aliases (Press 'D' to delete, 'Q' to go back)", true, tview.AlignCenter, tcell.ColorYellow)

	pages.AddPage("aliasList", frame, true, true)
	pages.SwitchToPage("aliasList")

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q', 'Q':
				pages.SwitchToPage("aliasManagement")
				app.SetInputCapture(nil)
				return nil
			case 'd', 'D':
				row, _ := table.GetSelection()
				if row > 0 {
					aliasToDelete := aliases[row-1]
					deleteAlias(app, pages, aliasFilePath, aliasToDelete.Name)
					return nil
				}
			}
		}
		return event
	})
}

func deleteAlias(app *tview.Application, pages *tview.Pages, aliasFilePath, name string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete the alias '%s'?", name)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				err := removeAlias(aliasFilePath, name)
				if err != nil {
					showErrorModal(app, pages, "Error deleting alias: "+err.Error())
				} else {
					listAliases(app, pages, aliasFilePath)
				}
			} else {
				pages.SwitchToPage("aliasList")
			}
		})

	pages.AddPage("deleteConfirm", modal, false, true)
	pages.SwitchToPage("deleteConfirm")
}

func checkInstallation(app *tview.Application, pages *tview.Pages, aliasFilePath, shellConfigPath string) {
	modal := tview.NewModal()
	isInstalled := isAliasmanInstalled(aliasFilePath, shellConfigPath)

	if isInstalled {
		modal.SetText("Aliasman is already installed.")
		modal.AddButtons([]string{"OK"})
	} else {
		modal.SetText("Aliasman is not installed. Would you like to install it?")
		modal.AddButtons([]string{"Install", "Cancel"})
	}

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if !isInstalled && buttonLabel == "Install" {
			installAliasman(aliasFilePath, shellConfigPath)
			modal.SetText("Aliasman has been installed successfully.")
			modal.ClearButtons()
			modal.AddButtons([]string{"OK"})
			app.SetFocus(modal)
		} else {
			pages.SwitchToPage("main")
		}
	})

	pages.AddPage("modal", modal, false, true)
	pages.SwitchToPage("modal")
}

func isAliasmanInstalled(aliasFilePath, shellConfigPath string) bool {
	// Check if alias file exists
	if _, err := os.Stat(aliasFilePath); os.IsNotExist(err) {
		return false
	}

	// Check if source line is in shell config
	content, err := os.ReadFile(shellConfigPath)
	if err != nil {
		return false
	}

	return string(content) != "" && (string(content) != "" && string(content) != "")
}

func installAliasman(aliasFilePath, shellConfigPath string) {
	// Create alias file
	if err := os.WriteFile(aliasFilePath, []byte("# Aliasman managed aliases\n"), 0644); err != nil {
		fmt.Println("Error creating alias file:", err)
		return
	}

	// Add source line to shell config
	sourceLine := fmt.Sprintf("\n%s\nsource %s\n%s\n", tagStart, aliasFilePath, tagEnd)
	f, err := os.OpenFile(shellConfigPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening shell config file:", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(sourceLine); err != nil {
		fmt.Println("Error writing to shell config file:", err)
	}
}

func addAlias(app *tview.Application, pages *tview.Pages, aliasFilePath string) {
	form := tview.NewForm()
	form.AddInputField("Alias Name", "", 20, nil, nil)
	form.AddInputField("Command", "", 50, nil, nil)
	form.AddButton("Save", func() {
		aliasName := form.GetFormItem(0).(*tview.InputField).GetText()
		command := form.GetFormItem(1).(*tview.InputField).GetText()

		if aliasName == "" || command == "" {
			showErrorModal(app, pages, "Both fields are required")
			return
		}

		err := appendAlias(aliasFilePath, aliasName, command)
		if err != nil {
			showErrorModal(app, pages, "Error adding alias: "+err.Error())
		} else {
			pages.SwitchToPage("aliasManagement")
		}
	}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("aliasManagement")
		})

	form.SetBorder(true).SetTitle("Add Alias").SetTitleAlign(tview.AlignCenter)
	form.SetButtonsAlign(tview.AlignCenter)

	pages.AddPage("addAlias", form, true, true)
	pages.SwitchToPage("addAlias")

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q') {
			pages.SwitchToPage("aliasManagement")
			app.SetInputCapture(nil)
			return nil
		}
		return event
	})
}

func showErrorModal(app *tview.Application, pages *tview.Pages, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("aliasManagement")
		})

	pages.AddPage("errorModal", modal, false, true)
	pages.SwitchToPage("errorModal")
}

type Alias struct {
	Name    string
	Command string
}

func readAliases(aliasFilePath string) ([]Alias, error) {
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	aliases := []Alias{}

	for _, line := range lines {
		if strings.HasPrefix(line, "alias ") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				command := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases = append(aliases, Alias{Name: name, Command: command})
			}
		}
	}

	return aliases, nil
}

func appendAlias(aliasFilePath, name, command string) error {
	f, err := os.OpenFile(aliasFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	aliasLine := fmt.Sprintf("alias %s='%s'\n", name, command)
	_, err = f.WriteString(aliasLine)
	return err
}

func removeAlias(aliasFilePath, name string) error {
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}

	for _, line := range lines {
		if !strings.HasPrefix(line, fmt.Sprintf("alias %s=", name)) {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(aliasFilePath, []byte(strings.Join(newLines, "\n")), 0644)
}

func detectShellConfig(homeDir string) string {
	shells := []string{".bashrc", ".zshrc", ".bash_profile"}
	for _, shell := range shells {
		path := filepath.Join(homeDir, shell)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func showAIAssistedAliasCreation(app *tview.Application, pages *tview.Pages, aliasFilePath string) {
	if !isLLMAvailable() {
		showErrorModal(app, pages, "The 'llm' command is not available on your system.")
		return
	}

	form := tview.NewForm()
	form.AddInputField("Describe the alias you want to create", "", 50, nil, nil)
	form.AddButton("Generate", func() {
		description := form.GetFormItem(0).(*tview.InputField).GetText()
		if description == "" {
			showErrorModal(app, pages, "Please enter a description for the alias.")
			return
		}
		generateAIAssistedAlias(app, pages, aliasFilePath, description)
	}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("main")
		})

	form.SetBorder(true).SetTitle("AI Assisted Alias Creation").SetTitleAlign(tview.AlignCenter)
	form.SetButtonsAlign(tview.AlignCenter)

	pages.AddPage("aiAssistedAlias", form, true, true)
	pages.SwitchToPage("aiAssistedAlias")
}

func isLLMAvailable() bool {
	cmd := exec.Command("llm", "--version")
	return cmd.Run() == nil
}

func generateAIAssistedAlias(app *tview.Application, pages *tview.Pages, aliasFilePath, description string) {
	cmd := exec.Command("llm", "-m", "llama3:8b", fmt.Sprintf("generate alias for %s, output just the command, as a bash command alias, inside a code block", description))
	output, err := cmd.CombinedOutput()
	if err != nil {
		showErrorModal(app, pages, fmt.Sprintf("Error generating alias: %v", err))
		return
	}

	alias := extractAliasFromOutput(string(output))
	if alias == "" {
		showAIOutput(app, pages, string(output))
		return
	}

	showAliasConfirmation(app, pages, aliasFilePath, alias)
}

func extractAliasFromOutput(output string) string {
	re := regexp.MustCompile("```(?:bash)?\n(alias .+?='.+')\n```")
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func showAIOutput(app *tview.Application, pages *tview.Pages, output string) {
	textView := tview.NewTextView().
		SetText(output).
		SetScrollable(true).
		SetDynamicColors(true)

	frame := tview.NewFrame(textView).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("AI Output (Press 'q' to go back)", true, tview.AlignCenter, tcell.ColorYellow)

	pages.AddPage("aiOutput", frame, true, true)
	pages.SwitchToPage("aiOutput")

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q') {
			pages.SwitchToPage("aiAssistedAlias")
			app.SetInputCapture(nil)
			return nil
		}
		return event
	})
}

func showAliasConfirmation(app *tview.Application, pages *tview.Pages, aliasFilePath, alias string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Do you want to add this alias?\n\n%s", alias)).
		AddButtons([]string{"Add", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Add" {
				parts := strings.SplitN(alias, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimPrefix(strings.TrimSpace(parts[0]), "alias ")
					command := strings.Trim(strings.TrimSpace(parts[1]), "'")
					err := appendAlias(aliasFilePath, name, command)
					if err != nil {
						showErrorModal(app, pages, "Error adding alias: "+err.Error())
					} else {
						pages.SwitchToPage("main")
					}
				} else {
					showErrorModal(app, pages, "Invalid alias format")
				}
			} else {
				pages.SwitchToPage("aiAssistedAlias")
			}
		})

	pages.AddPage("aliasConfirmation", modal, false, true)
	pages.SwitchToPage("aliasConfirmation")
}
