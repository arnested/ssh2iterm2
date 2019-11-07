package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	uuid "github.com/satori/go.uuid"
	"github.com/youtube/vitess/go/ioutil2"
)

type trigger struct {
	Partial   bool   `json:"partial"`
	Parameter string `json:"parameter"`
	Regex     string `json:"regex"`
	Action    string `json:"action"`
}

type profile struct {
	Badge         string `json:"Badge Text"`
	GUID          string `json:"Guid"`
	Name          string
	Command       string
	CustomCommand string       `json:"Custom Command"`
	Triggers      *triggerlist `json:",omitempty"`
	Tags          []string     `json:",omitempty"`
}

type triggerlist []*trigger

type profilelist struct {
	Profiles []*profile `json:",omitempty"`
}

func main() {
	ns, err := uuid.FromString("CAAFD038-5E80-4266-B6CF-F4D036E092F4")

	if err != nil {
		panic(err)
	}

	glob, present := os.LookupEnv("SSH2ITERM2_GLOB")

	if !present {
		userHomeDir, err := os.UserHomeDir()

		if err != nil {
			panic(err)
		}

		glob = userHomeDir + "/.ssh/config"
	}

	sshconfGlob, err := homedir.Expand(glob)

	if err != nil {
		panic(err)
	}

	files, err := filepath.Glob(sshconfGlob)

	if err != nil {
		panic(err)
	}

	ssh, err := exec.LookPath("ssh")
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile(`\*`)

	profiles := &profilelist{}

	for _, file := range files {
		processFile(file, r, ssh, ns, profiles)
	}

	json, err := json.MarshalIndent(profiles, "", "    ")

	if err != nil {
		panic(err)
	}

	userConfigDir, err := os.UserConfigDir()

	if err != nil {
		panic(err)
	}

	dynamicProfileFile, err := homedir.Expand(userConfigDir + "/iTerm2/DynamicProfiles/ssh2iterm2.json")

	if err != nil {
		panic(err)
	}

	err = ioutil2.WriteFileAtomic(dynamicProfileFile, json, 0644)
	if err != nil {
		panic(err)
	}
}

func processFile(file string, r *regexp.Regexp, ssh string, ns uuid.UUID, profiles *profilelist) {
	fileContent, err := os.Open(file)

	if err != nil {
		panic(err)
	}

	cfg, err := ssh_config.Decode(fileContent)

	if err != nil {
		panic(err)
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
			match := r.MatchString(name)
			if !match {
				uuid := uuid.NewV5(ns, name).String()
				profiles.Profiles = append(profiles.Profiles, &profile{
					Badge:         badge,
					GUID:          uuid,
					Name:          name,
					Command:       fmt.Sprintf("%s %s", ssh, hostname),
					CustomCommand: "Yes",
					Triggers: &triggerlist{&trigger{
						Partial:   true,
						Parameter: hostname,
						Regex:     "\\[sudo\\] password for",
						Action:    "PasswordTrigger",
					}},
					Tags: []string{tag},
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
