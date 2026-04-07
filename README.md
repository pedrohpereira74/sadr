# :( sadr — snippets + architecture decision records

**Capture code with context. Because snippets without a "why" are actually sad.**

sadr is a CLI tool that blends a snippet manager with Architectural Decision Records (ADR). It helps you capture code snippets directly from your clipboard, a file, or git diffs, and guides you to detail the context, decisions, and trade-offs behind that code — or any custom fields your team requires.


## ✨ Features

- **AI-Powered (`--smart`)**: Automatically deduces titles, context, and decisions from your code snippet using your desired AI model (currently only gemini models).
- **Multiple Sources**: Read from `--clipboard`, a `--file`, or your latest `--diff`.
- **Customizable Schemas**: You define the questions. The `config.yaml` is fully configurable. Add custom `select`, `multiselect`, `list`, or `text` fields. Supports multiple configs per project — switch between them with `--config`.
- **HTML Export**: Export your knowledge base to a single standalone, styled HTML file (`sadr export`).
- **Git-friendly**: sadr is designed to be used in git repositories. It stores records as markdown files in a `.sadr` directory, which can be tracked by git.

## 🚀 Installation

sadr provides pre-compiled binaries for all major platforms. Choose your preferred method:

### 🌟 Native Script
The easiest way to install sadr on macOS or Linux is via the native installer script:
```bash
curl -sSfL https://raw.githubusercontent.com/pedrohpereira74/sadr/master/install.sh | sh
```

### 🍺 Homebrew (macOS / Linux)
```bash
brew install pedrohpereira74/homebrew-tap/sadr
```

### 🪟 Scoop (Windows)
```bash
scoop bucket add pedrohpereira74 https://github.com/pedrohpereira74/scoop-bucket
scoop install sadr
```

### 🐧 APT / Debian
Download the `.deb` file from the [Releases page](https://github.com/pedrohpereira74/sadr/releases) or install directly:
```bash
wget https://github.com/pedrohpereira74/sadr/releases/latest/download/sadr_linux_amd64.deb
sudo apt install ./sadr_linux_amd64.deb
```

### 🎩 RPM / Fedora & RedHat
```bash
sudo rpm -i https://github.com/pedrohpereira74/sadr/releases/latest/download/sadr_linux_amd64.rpm
```

### 🐹 Go Install (Source)
If you have Go installed:
```bash
go install github.com/pedrohpereira74/sadr@latest
```

## 📚 Quick Start

1. **Initialize a project vault**:
   ```bash
   sadr init
   ```
2. **Setup Global Config (for AI & Editor)**:
   ```bash
   sadr init --global
   ```
3. **Setup AI API Key (for AI-powered features)**:
   ```bash
   sadr config --set-api-key "your-key"
   ```
4. **Capture a snippet**:
   ```bash
   # From clipboard, with AI auto-fill
   sadr new --clipboard --smart

   # From git diff, with AI auto-fill
   sadr new --diff --smart
   ```
5. **Search and List**:
   ```bash
   sadr list --tags "architecture,api"
   sadr search "auth logic" --deep
   ```
6. **Export to HTML**:
   ```bash
   sadr export --all
   ```

## ⚙️ Configuration

sadr uses three configuration files:
1. **Project schemas config (`.sadr/configs/default-config.yaml`)**: Team-oriented, git-friendly scope. Shareable schemas and fields. Tracked by Git.
2. **Personal schemas config (`~/.sadr/configs/default-config.yaml`)**: The "Dear Diary" schema. Personal fields for your private snippet vault. Strictly local, never tracked.
3. **Global config (`~/.sadr/global-config.yaml`)**: Personal preferences (Default Editor, Default Language, AI Provider credentials). Never tracked.


## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
