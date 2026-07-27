# git-commit-gen

![git-commit-gen demo](./assets/app.png)

Generate git commit messages automatically using an LLM.

## How it works

- The tool looks at the diff between your current branch and the base branch (default: main)
- It sends the diff to an OpenAI-compatible LLM API
- The LLM generates a commit message based on the diff and your rules.md file
- The commit message is printed to stdout

## Installation

```bash
go install github.com/yvv4git/git-commit-gen@latest
```

After installation, create default config files:

```bash
git-commit-gen setup
```

This creates `config.toml` and `rules.md` in `~/.config/git_commit_gen/`.

## Usage

Run inside a git repository:

```bash
git-commit-gen gen
```

If you want to see message in stdout:

```bash
git-commit-gen gen
```

To preview the generated message before applying:

```bash
git-commit-gen gen -v
```

## Configuration

Create a TOML config file (see configs/generator.example.toml):

- **generator.baseBranch** -- branch to diff against (default: main)
- **generator.rulesFile** -- path to rules.md with commit style rules
- **llm.openai.api** -- OpenAI-compatible API URL
- **llm.openai.token** -- API token
- **llm.openai.model** -- model name (default: gpt-4o)
- **llm.proxy** -- optional proxy settings (http or socks5)

Environment variables (OPENAI_TOKEN, OPENAI_API, OPENAI_MODEL, PROXY_TYPE, etc.) can also be used.

## Rules file

The rules.md file contains guidelines for commit message format. The LLM will follow these rules when generating messages. See rules.example.md for an example.

<p align="center">
  <a href="https://tonviewer.com/UQCcbp-mue-7HTjDNQ_ZrKtg-tUxIFu817APmItjXasiBGP3">
    <img src="https://img.shields.io/badge/Buy%20me%20a%20TON-0098EA?style=for-the-badge">
  </a>
</p>

<p align="center">
  If this tool helps you, consider buying me a coffee! ☕
</p>
