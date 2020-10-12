package runner

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/CircleCI-Public/circleci-cli/api/runner"
)

func generateConfig(t runner.Token, platform string, w io.Writer) (err error) {
	run := createRunnerNode(t)
	root := createRootNode(t, run)

	switch platform {
	default:
		return fmt.Errorf("unknown platform: %q", platform)

	case "minimal":

	case "linux":
		addLinuxConfig(run)

	case "macos":
		addMacConfig(run, root)
	}

	return yaml.NewEncoder(w).Encode(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{root},
	})
}

func createRootNode(t runner.Token, run *yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "api",
			},
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: "auth_token",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.FlowStyle,
						Value: t.Token,
					},
				},
			},

			{
				Kind:  yaml.ScalarNode,
				Value: "runner",
			},
			run,
		},
	}
}

func createRunnerNode(t runner.Token) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:        yaml.ScalarNode,
				Value:       "name",
				HeadComment: "# A friendly name to refer to this runner instance as",
			},
			{
				Kind:  yaml.ScalarNode,
				Style: yaml.DoubleQuotedStyle,
				Value: t.Nickname,
			},
		},
	}
}

func addLinuxConfig(run *yaml.Node) {
	run.Content = append(run.Content,
		[]*yaml.Node{
			{
				Kind:        yaml.ScalarNode,
				Value:       "command_prefix",
				HeadComment: "# This refers to the the systemd launch-wrapper the install instructions recommend",
			},
			{
				Kind:  yaml.SequenceNode,
				Style: yaml.FlowStyle,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "/opt/circleci/launch-task",
					},
				},
			},

			{
				Kind:        yaml.ScalarNode,
				Value:       "working_directory",
				HeadComment: "# Auto-generated working directory for jobs, can be automatically deleted",
			},
			{
				Kind:  yaml.ScalarNode,
				Style: yaml.DoubleQuotedStyle,
				Value: "/opt/circleci/workdir/%s",
			},

			{
				Kind:        yaml.ScalarNode,
				Value:       "cleanup_working_directory",
				HeadComment: "# Automatically delete the job working directories",
			},
			{
				Kind:  yaml.ScalarNode,
				Style: yaml.FlowStyle,
				Value: "true",
			},
		}...)
}

func addMacConfig(run *yaml.Node, root *yaml.Node) {
	run.Content = append(run.Content,
		[]*yaml.Node{
			{
				Kind:        yaml.ScalarNode,
				Value:       "command_prefix",
				HeadComment: "# Action required: Set the USERNAME",
			},
			{
				Kind:  yaml.SequenceNode,
				Style: yaml.FlowStyle,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "sudo",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "-niHu",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "USERNAME",
					},
					{
						Kind:  yaml.ScalarNode,
						Style: yaml.DoubleQuotedStyle,
						Value: "--",
					},
				},
			},

			{
				Kind:        yaml.ScalarNode,
				Value:       "working_directory",
				HeadComment: "# The working directory",
			},
			{
				Kind:  yaml.ScalarNode,
				Style: yaml.DoubleQuotedStyle,
				Value: "/tmp/%s",
			},

			{
				Kind:        yaml.ScalarNode,
				Value:       "cleanup_working_directory",
				HeadComment: "# Automatically delete the job working directories",
			},
			{
				Kind:  yaml.ScalarNode,
				Style: yaml.FlowStyle,
				Value: "true",
			},
		}...)

	root.Content = append(root.Content, []*yaml.Node{
		{
			Kind:  yaml.ScalarNode,
			Value: "logging",
		},
		{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{
					Kind:        yaml.ScalarNode,
					Value:       "file",
					HeadComment: "# Write runner logs to this file, logs will automatically be rotated by runner",
				},
				{
					Kind:  yaml.ScalarNode,
					Style: yaml.DoubleQuotedStyle,
					Value: "/Library/Logs/com.circleci.runner.log",
				},
			},
		},
	}...)
}
