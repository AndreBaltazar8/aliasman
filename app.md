Create a TUI application in Go

This TUI application is going to manage Bash alias. It should install a simple bash file in the home of the user, detect which shell the user is using, and check if a source to the bash file controlled by the aliasman is there. It should insert that within a tag, so it can be detected, installed/updated/removed.

The first option of the TUI, is it detected if the file is installed or not, and allows an option to install.

After its installed it changes options to manage the alias, allowing to list alias, and to add or delete

Maybe use @https://github.com/derailed/tview or something else for this TUI