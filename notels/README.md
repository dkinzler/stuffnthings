# What is this?

An experiment in using a language server to implement features of note taking apps.
Instead of using an app like Obsidian, you could use this server combined with any editor that supports the language server protocol.

Think of this as a proof of concept, right now only two simple features are implemented:
* being able to follow Obsidian-style markdown links (as the LSP GotoDefinition request)
* showing for a file the list of files that link to it (as the LSP FindReferences request)

### Neovim integration

Add the following to your Neovim config to autostart the server for `.md` files.
Note that you need a `.notes` file in one of the parent directories. The directory this file is in will be used as the project root for the server.

```lua
vim.api.nvim_create_autocmd("BufRead", {
    group = vim.api.nvim_create_augroup("notels", { clear = true }),
    pattern = { "*.md" },
    callback = function()
        vim.lsp.start({
            name = 'notels',
            cmd = { '/path/to/notels', '--logFile', 'lsp.log' },
            root_dir = vim.fs.dirname(vim.fs.find({ '.notes' }, { upward = true })[1]),
        })
    end
})
```
