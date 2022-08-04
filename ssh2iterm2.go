package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/gops/agent"
	"github.com/google/uuid"
	sshConfig "github.com/kevinburke/ssh_config"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"
	"github.com/rjeczalik/notify"
	"github.com/urfave/cli"
	"github.com/urfave/cli/altsrc"
	"github.com/youtube/vitess/go/ioutil2"
	"gopkg.in/yaml.v3"
)

type trigger struct {
	Partial   bool   `json:"partial"`
	Parameter string `json:"parameter"`
	Regex     string `json:"regex"`
	Action    string `json:"action"`
}

//nolint:tagliatelle
type profile struct {
	Badge         string `json:"Badge Text"`
	GUID          string `json:"Guid"`
	Name          string
	Command       string
	CustomCommand string       `json:"Custom Command"`
	Triggers      *triggerlist `json:",omitempty"`
	Tags          []string     `json:",omitempty"`
	BoundHosts    []string     `json:"Bound Hosts,omitempty"`
}

type triggerlist []*trigger

//nolint:tagliatelle
type profilelist struct {
	Profiles []*profile `json:",omitempty"`
}

// GitSummary is the version string to be set at compile time via command line.
var GitSummary string //nolint:gochecknoglobals

//nolint:funlen // needs refactoring.
func main() {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.SetFlags(0)
	}

	app := cli.NewApp()
	app.Name = "ssh2iterm2"
	app.Usage = "Create iTerm2 dynamic profile from SSH config"
	app.EnableBashCompletion = true
	app.Authors = []cli.Author{
		{
			Name:  "Arne Jørgensen",
			Email: "arne@arnested.dk",
		},
	}
	app.Version = GitSummary

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		userHomeDir = "~/"
	}

	ssh, err := exec.LookPath("ssh")
	if err != nil {
		ssh = "ssh"
	}

	configPath := ""
	userConfigDir, err := os.UserConfigDir()

	if err == nil {
		configPath = userConfigDir + "/ssh2iterm2.yaml"
	}

	app.Flags = []cli.Flag{
		altsrc.NewStringFlag(cli.StringFlag{
			Name:      "glob",
			Value:     userHomeDir + "/.ssh/config",
			Usage:     "A file `GLOB` matching ssh config file(s)",
			EnvVar:    "SSH2ITERM2_GLOB",
			TakesFile: true,
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name:      "ssh",
			Value:     ssh,
			Usage:     "The ssh client `PATH`",
			EnvVar:    "SSH2ITERM2_SSH_PATH",
			TakesFile: true,
		}),
		altsrc.NewBoolFlag(cli.BoolFlag{
			Name:   "automatic-profile-switching",
			Usage:  "Add hostname for automatic profile switching`",
			EnvVar: "SSH2ITERM2_AUTOMATIC_PROFILE_SWITCHING",
		}),
		cli.StringFlag{
			Name:      "config",
			Value:     configPath,
			Usage:     "Read config from `FILE`",
			EnvVar:    "SSH2ITERM2_CONFIG_FILE",
			TakesFile: true,
		},
		altsrc.NewBoolFlag(cli.BoolFlag{
			Name:   "enable-gops-agent",
			Usage:  "Run with a gops agent (see https://pkg.go.dev/github.com/google/gops?tab=overview)",
			EnvVar: "SSH2ITERM2_WITH_GOPS_AGENT",
		}),
	}

	app.Before = func(ctx *cli.Context) error {
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			initConfig := altsrc.InitInputSourceWithContext(app.Flags, altsrc.NewYamlSourceFromFlagFunc("config"))
			_ = initConfig(ctx)
		}

		if ctx.GlobalBool("enable-gops-agent") {
			if err := agent.Listen(agent.Options{ShutdownCleanup: true}); err != nil {
				log.Fatal(err)
			}
		}

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:   "sync",
			Usage:  "Sync ssh config to iTerm2 dynamic profiles",
			Action: ssh2iterm2,
		},
		{
			Name:   "watch",
			Usage:  "Continuously watch and sync folder for changes",
			Action: watch,
		},
		{
			Name:   "edit-config",
			Usage:  "Edit config file",
			Action: editConfig,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "editor",
					Value:  "vi",
					Usage:  "Use `EDITOR` to edit config file (create it of it doesn't exist)",
					EnvVar: "EDITOR",
				},
			},
		},
	}

	err = app.Run(os.Args)

	if err != nil {
		log.Fatal(err)
	}
}

func ssh2iterm2(ctx *cli.Context) error {
	namespace, err := uuid.Parse("CAAFD038-5E80-4266-B6CF-F4D036E092F4")
	if err != nil {
		return fmt.Errorf("failed to parse static uuid: %w", err)
	}

	glob, err := homedir.Expand(ctx.GlobalString("glob"))
	if err != nil {
		return fmt.Errorf("failed to expand glob: %w", err)
	}

	log.Printf("Glob is %q", glob)

	files, err := filepath.Glob(glob)
	if err != nil {
		return fmt.Errorf("failed to get files matching glob: %w", err)
	}

	regex := regexp.MustCompile(`\*`)

	profiles := &profilelist{}

	automaticProfileSwitching := ctx.GlobalBool("automatic-profile-switching")
	ssh := ctx.GlobalString("ssh")
	log.Printf("SSH cli is %q", ssh)

	for _, file := range files {
		processFile(file, regex, ssh, namespace, profiles, automaticProfileSwitching)
	}

	json, err := json.MarshalIndent(profiles, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles into JSON: %w", err)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to locate user config directory: %w", err)
	}

	dynamicProfileFile, err := homedir.Expand(userConfigDir + "/iTerm2/DynamicProfiles/ssh2iterm2.json")
	if err != nil {
		return fmt.Errorf("failed to expand name of dynamic profile file: %w", err)
	}

	log.Printf("Writing %q", dynamicProfileFile)

	//nolint:gomnd
	err = ioutil2.WriteFileAtomic(dynamicProfileFile, json, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write dynamic profile file: %w", err)
	}

	return nil
}

//nolint:funlen // needs refactoring.
func processFile(file string,
	regex *regexp.Regexp,
	ssh string,
	namespace uuid.UUID,
	profiles *profilelist,
	automaticProfileSwitching bool,
) {
	log.Printf("Parsing %q", file)

	fileContent, err := os.Open(file)
	if err != nil {
		log.Print(err)

		return
	}

	cfg, err := sshConfig.Decode(fileContent)
	if err != nil {
		log.Print(err)

		return
	}

	tag := tag(file)

	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			hostname := pattern.String()
			name := hostname
			badge := hostname
			comment := strings.TrimSpace(host.EOLComment)

			if comment != "" {
				badge = comment
				name = fmt.Sprintf("%s (%s)", hostname, comment)
			}

			match := regex.MatchString(name)
			if !match {
				uuid := uuid.NewSHA1(namespace, []byte(name)).String()
				log.Printf("Identified %s", name)

				var boundHosts []string
				if automaticProfileSwitching {
					boundHosts = []string{hostname}
				}

				profiles.Profiles = append(profiles.Profiles, &profile{
					Badge:         badge,
					GUID:          uuid,
					Name:          name,
					Command:       fmt.Sprintf("%q %q", ssh, hostname),
					CustomCommand: "Yes",
					Triggers: &triggerlist{&trigger{
						Partial:   true,
						Parameter: hostname,
						Regex:     "\\[sudo\\] password for",
						Action:    "PasswordTrigger",
					}},
					Tags:       []string{tag},
					BoundHosts: boundHosts,
				})
			}
		}
	}
}

func tag(filename string) string {
	base := path.Base(strings.TrimSuffix(filename, path.Ext(filename)))
	re := regexp.MustCompile(`^[0-9]+_`)

	return re.ReplaceAllString(base, `$1`)
}

const channelBufferSize = 10

func watch(ctx *cli.Context) error {
	glob, err := homedir.Expand(ctx.GlobalString("glob"))
	if err != nil {
		return fmt.Errorf("failed to expand glob: %w", err)
	}

	//nolint:gomnd
	dir := filepath.Dir(strings.SplitAfterN(glob, "*", 2)[0])
	log.Printf("Watching is %q", dir)

	eventChan := make(chan notify.EventInfo, channelBufferSize)

	if err := notify.Watch(dir+"/...", eventChan, notify.All); err != nil {
		log.Fatal(err)
	}

	defer notify.Stop(eventChan)

	for {
		eventInfo := <-eventChan
		if match, err := filepath.Match(glob, eventInfo.Path()); err == nil && match {
			_ = ssh2iterm2(ctx)
		}
	}
}

type config struct {
	Glob string `yaml:"glob"`
	SSH  string `yaml:"ssh"`
}

func editConfig(ctx *cli.Context) error {
	configFile := ctx.GlobalString("config")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err := createConfig(configFile, config{
			Glob: ctx.GlobalString("glob"),
			SSH:  ctx.GlobalString("ssh"),
		})
		if err != nil {
			return err
		}
	}

	editCmd := ctx.String("editor") + " '" + configFile + "'"
	cmd := exec.Command("sh", "-c", editCmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor command: %w", err)
	}

	return nil
}

func createConfig(configFile string, config config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config into YAML: %w", err)
	}

	//nolint:gomnd
	err = ioutil2.WriteFileAtomic(configFile, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
