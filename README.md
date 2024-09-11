# Aliasman

![Aliasman Logo](path/to/logo.png)

Aliasman is a powerful Terminal User Interface (TUI) application for managing Bash aliases with ease. It simplifies the process of creating, listing, and deleting aliases, and even offers AI-assisted alias creation.

## Features

- 🚀 Easy installation and setup
- 📋 List, add, and delete aliases
- 🤖 AI-assisted alias creation
- ⚙️ Configurable LLM model for AI assistance
- 🖥️ Cross-shell compatibility (Bash, Zsh)
- 🎨 User-friendly TUI powered by tview

## Installation

### Prerequisites

- Go 1.21 or higher
- [LLM](https://llm.datasette.io/en/stable/) (for AI-assisted alias creation)

You can choose either the quick install method or the manual installation steps below.

### Quick Install

To quickly install Aliasman globally, use the following command:

```
go install github.com/AndreBaltazar8/aliasman@latest
```

### Steps

1. Clone the repository:
   ```
   git clone https://github.com/AndreBaltazar8/aliasman.git
   ```

2. Navigate to the project directory:
   ```
   cd aliasman
   ```

3. Install the application:
   ```
   go install
   ```

4. Run Aliasman:
   ```
   aliasman
   ```

## Usage

1. Launch Aliasman by running `aliasman` in your terminal.
2. Use the arrow keys to navigate the menu and Enter to select an option.
3. Follow the on-screen prompts to manage your aliases.

### Main Menu Options

- **Manage Aliases**: Add, remove, or list aliases
- **AI Assisted Alias Creation**: Create aliases with AI help
- **Settings**: Configure Aliasman and check installation
- **Quit**: Exit the application

### Quick Alias Listing

To quickly list all aliases without entering the TUI:

```
aliasman list
```

## Configuration

Aliasman stores its configuration and aliases in `~/.aliasman_aliases`. You can manually edit this file, but it's recommended to use the TUI for management.

To change the LLM model used for AI-assisted alias creation, use the "Change LLM Model" option in the Settings menu.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).