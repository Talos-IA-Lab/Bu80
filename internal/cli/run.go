package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"bu80/internal/agent"
	"bu80/internal/config"
	"bu80/internal/contextcmd"
	"bu80/internal/loop"
	"bu80/internal/state"
	"bu80/internal/statuscmd"
	"bu80/internal/taskcmd"
)

var Version = "dev"

var (
	stdout  = os.Stdout
	stderr  = os.Stderr
	nowFunc = time.Now
)

type Options struct {
	Agent             string
	Model             string
	PromptFile        string
	PromptTemplate    string
	CompletionPromise string
	AbortPromise      string
	TaskPromise       string
	Rotation          string
	TasksMode         bool
	MinIterations     int
	MaxIterations     int
	Status            bool
	ListTasks         bool
	AddTask           string
	RemoveTask        int
	AddContext        string
	ClearContext      bool
	InitConfig        bool
	Config            string
	NoStream          bool
	Stream            bool
	VerboseTools      bool
	Questions         bool
	NoQuestions       bool
	NoPlugins         bool
	NoCommit          bool
	AllowAll          bool
	NoAllowAll        bool
	ShowVersion       bool
	ShowHelp          bool
	ExtraArgs         []string
	PromptArgs        []string
}

func Run(args []string) error {
	opts, fs, err := Parse(args)
	if err != nil {
		return err
	}

	switch {
	case opts.ShowHelp:
		return printHelp(stdout, fs)
	case opts.ShowVersion:
		_, err := fmt.Fprintln(stdout, Version)
		return err
	case opts.Status:
		snapshot, err := statuscmd.LoadSnapshot(nowFunc())
		if err != nil {
			return err
		}
		return statuscmd.Write(stdout, snapshot, opts.TasksMode)
	case opts.ListTasks:
		return taskcmd.List(stdout)
	case opts.AddTask != "":
		return taskcmd.Add(opts.AddTask)
	case opts.RemoveTask > 0:
		return taskcmd.Remove(opts.RemoveTask)
	case opts.AddContext != "":
		return contextcmd.Add(opts.AddContext, nowFunc())
	case opts.ClearContext:
		return contextcmd.Clear(stdout)
	case opts.InitConfig:
		path, err := config.InitDefaultConfig(opts.Config)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, path)
		return err
	}

	cfg, err := config.Load(opts.Config)
	if err != nil {
		return err
	}

	questionsEnabled := true
	if cfg.QuestionsEnabled != nil {
		questionsEnabled = *cfg.QuestionsEnabled
	}
	if opts.Questions {
		questionsEnabled = true
	}
	if opts.NoQuestions {
		questionsEnabled = false
	}

	streamEnabled := true
	if opts.NoStream {
		streamEnabled = false
	}
	if opts.Stream {
		streamEnabled = true
	}

	return loop.Run(loop.Options{
		Agent:             opts.Agent,
		Model:             opts.Model,
		PromptFile:        opts.PromptFile,
		PromptArgs:        opts.PromptArgs,
		PromptTemplate:    opts.PromptTemplate,
		Rotation:          opts.Rotation,
		ConfigPath:        opts.Config,
		NoPlugins:         opts.NoPlugins,
		QuestionsEnabled:  questionsEnabled,
		VerboseTools:      opts.VerboseTools,
		CompletionPromise: opts.CompletionPromise,
		AbortPromise:      opts.AbortPromise,
		TaskPromise:       opts.TaskPromise,
		TasksMode:         opts.TasksMode,
		MinIterations:     opts.MinIterations,
		MaxIterations:     opts.MaxIterations,
		AllowAll:          opts.AllowAll,
		NoCommit:          opts.NoCommit,
		ExtraArgs:         opts.ExtraArgs,
		Stream:            streamEnabled,
		Stdout:            stdout,
		Stderr:            stderr,
		Stdin:             os.Stdin,
		Now:               nowFunc,
	})
}

func Parse(args []string) (Options, *flag.FlagSet, error) {
	var opts Options

	fs := flag.NewFlagSet("bu80", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&opts.Agent, "agent", "codex", "agent to use")
	fs.IntVar(&opts.MinIterations, "min-iterations", 1, "minimum iterations")
	fs.IntVar(&opts.MaxIterations, "max-iterations", 0, "maximum iterations")
	fs.StringVar(&opts.CompletionPromise, "completion-promise", state.DefaultDonePromise, "completion promise")
	fs.StringVar(&opts.AbortPromise, "abort-promise", "", "abort promise")
	fs.BoolVar(&opts.TasksMode, "tasks", false, "enable tasks mode")
	fs.BoolVar(&opts.TasksMode, "t", false, "enable tasks mode")
	fs.StringVar(&opts.TaskPromise, "task-promise", state.DefaultTaskPromise, "task promise")
	fs.StringVar(&opts.Model, "model", "", "model")
	fs.StringVar(&opts.Rotation, "rotation", "", "rotation list")
	fs.StringVar(&opts.PromptFile, "prompt-file", "", "prompt file")
	fs.StringVar(&opts.PromptFile, "file", "", "prompt file")
	fs.StringVar(&opts.PromptFile, "f", "", "prompt file")
	fs.StringVar(&opts.PromptTemplate, "prompt-template", "", "prompt template")
	fs.BoolVar(&opts.NoStream, "no-stream", false, "disable streaming")
	fs.BoolVar(&opts.Stream, "stream", false, "enable streaming")
	fs.BoolVar(&opts.VerboseTools, "verbose-tools", false, "verbose tool output")
	fs.BoolVar(&opts.Questions, "questions", false, "enable questions")
	fs.BoolVar(&opts.NoQuestions, "no-questions", false, "disable questions")
	fs.BoolVar(&opts.NoPlugins, "no-plugins", false, "disable plugins")
	fs.BoolVar(&opts.NoCommit, "no-commit", false, "disable auto-commit")
	fs.BoolVar(&opts.AllowAll, "allow-all", false, "allow all permissions")
	fs.BoolVar(&opts.NoAllowAll, "no-allow-all", false, "disable allow all")
	fs.BoolVar(&opts.ShowVersion, "version", false, "show version")
	fs.BoolVar(&opts.ShowVersion, "v", false, "show version")
	fs.BoolVar(&opts.ShowHelp, "help", false, "show help")
	fs.BoolVar(&opts.ShowHelp, "h", false, "show help")
	fs.BoolVar(&opts.Status, "status", false, "show status")
	fs.BoolVar(&opts.ListTasks, "list-tasks", false, "list tasks")
	fs.StringVar(&opts.AddTask, "add-task", "", "add task")
	fs.IntVar(&opts.RemoveTask, "remove-task", 0, "remove task")
	fs.StringVar(&opts.AddContext, "add-context", "", "add context")
	fs.BoolVar(&opts.ClearContext, "clear-context", false, "clear context")
	fs.StringVar(&opts.Config, "config", "", "config path")
	fs.BoolVar(&opts.InitConfig, "init-config", false, "init config")

	if err := fs.Parse(args); err != nil {
		return opts, fs, err
	}

	opts.PromptArgs = fs.Args()
	if idx := indexOfDoubleDash(args); idx >= 0 {
		opts.ExtraArgs = append([]string(nil), args[idx+1:]...)
	}

	if err := validate(opts); err != nil {
		return opts, fs, err
	}

	return opts, fs, nil
}

func validate(opts Options) error {
	if opts.MaxIterations != 0 && opts.MinIterations > opts.MaxIterations {
		return errors.New("min-iterations must not be greater than max-iterations")
	}
	if opts.TasksMode && strings.TrimSpace(opts.CompletionPromise) == strings.TrimSpace(opts.TaskPromise) {
		return errors.New("completion-promise and task-promise must differ in tasks mode")
	}
	if _, ok := agent.Builtins()[opts.Agent]; !ok {
		return fmt.Errorf("unknown agent: %s", opts.Agent)
	}
	if _, err := agent.ParseRotation(opts.Rotation, agent.Builtins()); err != nil {
		return err
	}
	return nil
}

func printHelp(w io.Writer, fs *flag.FlagSet) error {
	_, err := fmt.Fprintln(w, "usage: bu80 [options] <prompt>")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, "\nOptions:")
	if err != nil {
		return err
	}
	fs.SetOutput(w)
	fs.PrintDefaults()
	return nil
}

func indexOfDoubleDash(args []string) int {
	for i, arg := range args {
		if arg == "--" {
			return i
		}
	}
	return -1
}
