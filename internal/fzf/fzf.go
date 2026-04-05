package fzf

import (
	"fmt"
	"os/exec"
	"strings"
)

type FZFOptions struct {
	Prompt        string
	Preview       string
	PreviewWindow string
	Multi         bool
	HeaderLines   int
	Height        string
}

func NewOptions() *FZFOptions {
	return &FZFOptions{
		Multi:         false,
		PreviewWindow: "right:50%",
		Height:        "50%",
	}
}

func (o *FZFOptions) WithPrompt(prompt string) *FZFOptions {
	o.Prompt = prompt
	return o
}

func (o *FZFOptions) WithMulti() *FZFOptions {
	o.Multi = true
	return o
}

func (o *FZFOptions) WithPreview(preview string) *FZFOptions {
	o.Preview = preview
	return o
}

func (o *FZFOptions) WithPreviewWindow(window string) *FZFOptions {
	o.PreviewWindow = window
	return o
}

func (o *FZFOptions) WithHeight(height string) *FZFOptions {
	o.Height = height
	return o
}

func Select(items []string, opts *FZFOptions) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select")
	}

	args := []string{"--height=" + opts.Height}

	if opts.Prompt != "" {
		args = append(args, "--prompt="+opts.Prompt)
	}

	if opts.Multi {
		args = append(args, "--multi")
	}

	if opts.Preview != "" {
		args = append(args, "--preview="+opts.Preview)
	}

	if opts.PreviewWindow != "" {
		args = append(args, "--preview-window="+opts.PreviewWindow)
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return "", nil
			}
		}
		return "", err
	}

	selected := strings.TrimSpace(string(output))
	return selected, nil
}

func SelectMulti(items []string, opts *FZFOptions) ([]string, error) {
	opts.Multi = true
	selected, err := Select(items, opts)
	if err != nil {
		return nil, err
	}
	if selected == "" {
		return []string{}, nil
	}
	return strings.Split(selected, "\n"), nil
}

func EnsureFZFInstalled() error {
	_, err := exec.LookPath("fzf")
	if err != nil {
		return fmt.Errorf("fzf is not installed. Please install fzf first:\n  - Linux/macOS: sudo apt install fzf or brew install fzf\n  - Windows: choco install fzf or winget install fzf")
	}
	return nil
}
