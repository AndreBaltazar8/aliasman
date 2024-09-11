package main

import (
	"encoding/json"
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

	// Check if the "list" parameter is provided
	if len(os.Args) > 1 && os.Args[1] == "list" {
		listAliasesCli()
		return
	}

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
		AddItem("AI Assisted Alias Creation", "Create an alias using AI assistance", 'a', nil).
		AddItem("Settings", "Configure Aliasman settings", 's', nil).
		AddItem("Quit", "Exit the application", 'q', func() {
			app.Stop()
			showReloadInstructions(shellConfigPath)
		})

	mainMenu.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		switch index {
		case 0:
			showAliasManagement(app, pages, aliasFilePath)
		case 1:
			showAIAssistedAliasCreation(app, pages, aliasFilePath)
		case 2:
			showSettings(app, pages, aliasFilePath, shellConfigPath)
		}
	})

	return mainMenu
}

func showReloadInstructions(shellConfigPath string) {
	fmt.Printf("\nTo reload your aliases in the current shell, you can either:\n")
	fmt.Printf("1. Run the command: source %s\n", shellConfigPath)
	fmt.Printf("2. Or simply use the alias: aliasman-reload\n\n")
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

	table.SetCell(0, 0, tview.NewTableCell("Type").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	table.SetCell(0, 1, tview.NewTableCell("Name").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	table.SetCell(0, 2, tview.NewTableCell("Command").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))

	for i, alias := range aliases {
		table.SetCell(i+1, 0, tview.NewTableCell(alias.Type).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		table.SetCell(i+1, 1, tview.NewTableCell(alias.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		table.SetCell(i+1, 2, tview.NewTableCell(alias.Command).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
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
	// Create alias file with configuration
	initialContent := `# { "model": "llama3:8b" }
# Aliasman managed aliases

# Reload aliases
alias aliasman-reload='source ` + aliasFilePath + `'
`
	if err := os.WriteFile(aliasFilePath, []byte(initialContent), 0644); err != nil {
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
	form.AddInputField("Name", "", 20, nil, nil)
	form.AddInputField("Command", "", 50, nil, nil)
	form.AddDropDown("Type", []string{"alias", "function"}, 0, nil)
	form.AddButton("Save", func() {
		name := form.GetFormItem(0).(*tview.InputField).GetText()
		command := form.GetFormItem(1).(*tview.InputField).GetText()
		_, aliasType := form.GetFormItem(2).(*tview.DropDown).GetCurrentOption()

		if name == "" || command == "" {
			showErrorModal(app, pages, "Both fields are required")
			return
		}

		err := appendAlias(aliasFilePath, name, command, aliasType)
		if err != nil {
			showErrorModal(app, pages, "Error adding alias/function: "+err.Error())
		} else {
			pages.SwitchToPage("aliasManagement")
		}
	}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("aliasManagement")
		})

	form.SetBorder(true).SetTitle("Add Alias/Function").SetTitleAlign(tview.AlignCenter)
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
	Type    string // "alias" or "function"
}

func readAliases(aliasFilePath string) ([]Alias, error) {
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	aliases := []Alias{}
	inFunction := false
	currentFunction := Alias{}

	for _, line := range lines {
		if strings.HasPrefix(line, "alias ") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				command := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases = append(aliases, Alias{Name: name, Command: command, Type: "alias"})
			}
		} else if strings.HasPrefix(line, "function ") || strings.HasSuffix(line, "() {") {
			inFunction = true
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "function "), "() {"))
			currentFunction = Alias{Name: name, Command: "", Type: "function"}
		} else if inFunction {
			if line == "}" {
				inFunction = false
				aliases = append(aliases, currentFunction)
			} else {
				currentFunction.Command += line + "\n"
			}
		}
	}

	return aliases, nil
}

func appendAlias(aliasFilePath string, name, command, aliasType string) error {
	f, err := os.OpenFile(aliasFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var aliasLine string
	if aliasType == "alias" {
		aliasLine = fmt.Sprintf("alias %s='%s'\n", name, command)
	} else {
		aliasLine = fmt.Sprintf("function %s() {\n%s\n}\n", name, command)
	}
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
		showErrorModal(app, pages, "The 'llm' command is not available on your system. Install it: https://llm.datasette.io/en/stable/")
		return
	}

	form := tview.NewForm()
	form.AddDropDown("Type", []string{"Alias", "Function"}, 0, nil)
	form.AddInputField("Description", "", 50, nil, nil)
	form.AddButton("Generate", func() {
		_, typeStr := form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()
		description := form.GetFormItem(1).(*tview.InputField).GetText()
		if description == "" {
			showErrorModal(app, pages, "Please enter a description.")
			return
		}
		generateAIAssistedAliasOrFunction(app, pages, aliasFilePath, typeStr, description)
	}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("main")
		})

	form.SetBorder(true).SetTitle("AI Assisted Alias/Function Creation").SetTitleAlign(tview.AlignCenter)
	form.SetButtonsAlign(tview.AlignCenter)

	pages.AddPage("aiAssistedCreation", form, true, true)
	pages.SwitchToPage("aiAssistedCreation")
}

func generateAIAssistedAliasOrFunction(app *tview.Application, pages *tview.Pages, aliasFilePath, typeStr, description string) {
	config, err := readConfig(aliasFilePath)
	if err != nil {
		showErrorModal(app, pages, fmt.Sprintf("Error reading configuration: %v", err))
		return
	}

	var prompt string
	if typeStr == "Alias" {
		prompt = fmt.Sprintf("generate alias for %s, output just the command, as a bash command alias, inside a code block", description)
	} else {
		prompt = fmt.Sprintf("generate function for %s, output just the function, as a bash function (with function prefix), inside a code block", description)
	}

	cmd := exec.Command("llm", "-m", config.Model, prompt)
	output, err := cmd.CombinedOutput()
	if err != nil {
		showErrorModal(app, pages, fmt.Sprintf("Error generating %s: %v", strings.ToLower(typeStr), err))
		return
	}

	result := extractAliasOrFunctionFromOutput(string(output))
	if result == "" {
		showAIOutput(app, pages, string(output))
		return
	}

	showAliasOrFunctionConfirmation(app, pages, aliasFilePath, result)
}

func extractAliasOrFunctionFromOutput(output string) string {
	re := regexp.MustCompile("(?s).*```(?:bash)?\n((?:alias .+?='.+')|(?:function .+? \\{[\\s\\S]+?\\}))\n```.*")
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func showAliasOrFunctionConfirmation(app *tview.Application, pages *tview.Pages, aliasFilePath, result string) {
	isFunction := strings.HasPrefix(result, "function")
	typeStr := "alias"
	if isFunction {
		typeStr = "function"
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Do you want to add this %s?\n\n%s", typeStr, result)).
		AddButtons([]string{"Add", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Add" {
				var name, command string
				if isFunction {
					parts := strings.SplitN(result, "{", 2)
					name = strings.TrimSpace(strings.TrimPrefix(parts[0], "function"))
					name = strings.TrimSuffix(name, "()")
					command = "{" + parts[1]
				} else {
					parts := strings.SplitN(result, "=", 2)
					name = strings.TrimPrefix(strings.TrimSpace(parts[0]), "alias ")
					command = strings.Trim(strings.TrimSpace(parts[1]), "'")
				}

				err := appendAlias(aliasFilePath, name, command, typeStr)
				if err != nil {
					showErrorModal(app, pages, fmt.Sprintf("Error adding %s: %v", typeStr, err))
				} else {
					pages.SwitchToPage("main")
				}
			} else {
				pages.SwitchToPage("aiAssistedCreation")
			}
		})

	pages.AddPage("aliasOrFunctionConfirmation", modal, false, true)
	pages.SwitchToPage("aliasOrFunctionConfirmation")
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
			pages.SwitchToPage("aiAssistedCreation")
			app.SetInputCapture(nil)
			return nil
		}
		return event
	})
}

func isLLMAvailable() bool {
	cmd := exec.Command("llm", "--version")
	return cmd.Run() == nil
}

type Config struct {
	Model string `json:"model"`
}

func readConfig(aliasFilePath string) (Config, error) {
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return Config{}, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# {") && strings.HasSuffix(line, "}") {
			var config Config
			err := json.Unmarshal([]byte(line[2:]), &config)
			if err == nil {
				return config, nil
			}
		}
	}

	return Config{Model: "llama3:8b"}, nil // Default model if not found
}

func listAliasesCli() {
	aliases, functions, err := loadAliasesAndFunctions()
	if err != nil {
		fmt.Println("Error loading aliases and functions:", err)
		return
	}

	fmt.Println("Available aliases:")
	for name, command := range aliases {
		fmt.Printf("  %s: %s\n", name, command)
	}

	fmt.Println("\nAvailable functions:")
	for name := range functions {
		fmt.Printf("  %s\n", name)
	}
}

func loadAliasesAndFunctions() (map[string]string, map[string]bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting home directory: %w", err)
	}

	aliasFilePath := filepath.Join(homeDir, aliasFileName)
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading alias file: %w", err)
	}

	aliases := make(map[string]string)
	functions := make(map[string]bool)
	lines := strings.Split(string(content), "\n")

	inFunction := false
	currentFunction := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "alias ") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				command := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases[name] = command
			}
		} else if strings.HasPrefix(line, "function ") || strings.HasSuffix(line, "() {") {
			inFunction = true
			currentFunction = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "function "), "() {"))
			functions[currentFunction] = true
		} else if line == "}" && inFunction {
			inFunction = false
			currentFunction = ""
		}
	}

	return aliases, functions, nil
}

func showSettings(app *tview.Application, pages *tview.Pages, aliasFilePath, shellConfigPath string) {
	list := tview.NewList().
		AddItem("Check Installation", "Check if Aliasman is installed", 'c', nil).
		AddItem("Change LLM Model", "Modify the AI model used for alias generation", 'm', nil).
		AddItem("Back", "Return to main menu", 'q', nil)

	list.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		switch index {
		case 0:
			checkInstallation(app, pages, aliasFilePath, shellConfigPath)
		case 1:
			changeLLMModel(app, pages, aliasFilePath)
		case 2:
			pages.SwitchToPage("main")
		}
	})

	pages.AddPage("settings", list, true, true)
	pages.SwitchToPage("settings")
}

func changeLLMModel(app *tview.Application, pages *tview.Pages, aliasFilePath string) {
	config, err := readConfig(aliasFilePath)
	if err != nil {
		showErrorModal(app, pages, fmt.Sprintf("Error reading configuration: %v", err))
		return
	}

	// Run "llm models" command
	cmd := exec.Command("llm", "models")
	output, err := cmd.CombinedOutput()
	if err != nil {
		showErrorModal(app, pages, fmt.Sprintf("Error getting available models: %v", err))
		return
	}

	// Create a flex layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(tcell.ColorBlack)

	// Add a text view for the model list
	modelList := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetText(string(output))

	flex.AddItem(tview.NewTextView().SetText("Available Models (scrollable)").SetTextAlign(tview.AlignCenter), 1, 1, false)
	flex.AddItem(modelList, 0, 1, false)

	// Add a form for changing the model
	form := tview.NewForm()
	form.AddInputField("LLM Model", config.Model, 30, nil, nil)
	form.AddButton("Save", func() {
		newModel := form.GetFormItem(0).(*tview.InputField).GetText()
		if newModel == "" {
			showErrorModal(app, pages, "Model name cannot be empty")
			return
		}

		config.Model = newModel
		err := updateConfig(aliasFilePath, config)
		if err != nil {
			showErrorModal(app, pages, fmt.Sprintf("Error updating configuration: %v", err))
		} else {
			pages.SwitchToPage("settings")
		}
	})
	form.AddButton("Cancel", func() {
		pages.SwitchToPage("settings")
	})

	flex.AddItem(form, 0, 1, true)

	// Create a frame to hold the flex layout
	frame := tview.NewFrame(flex).SetBorders(0, 0, 0, 0, 0, 0)
	frame.AddText("Change LLM Model", true, tview.AlignCenter, tcell.ColorYellow)

	pages.AddPage("changeLLMModel", frame, true, true)
	pages.SwitchToPage("changeLLMModel")

	// Set input capture for the entire page
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("settings")
			app.SetInputCapture(nil)
			return nil
		}
		return event
	})
}

func updateConfig(aliasFilePath string, config Config) error {
	content, err := os.ReadFile(aliasFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	newConfigLine := fmt.Sprintf("# %s", string(configJSON))
	updatedLines := []string{newConfigLine}

	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "# {") && strings.HasSuffix(line, "}") {
			continue
		}
		updatedLines = append(updatedLines, line)
	}

	return os.WriteFile(aliasFilePath, []byte(strings.Join(updatedLines, "\n")), 0644)
}
