package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
)

// JsonQuery plugin
type JsonQuery struct{}

// fatalIf will print the error and exit when err is not nil
func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: ", err)
		os.Exit(1)
	}
}

// extractAppName return the application name parameter from the command line
func (c *JsonQuery) extractAppName(args []string) (string, error) {
	if len(args) < 2 {
		return "", errors.New("missing application name")
	}

	return args[1], nil
}

func (c *JsonQuery) retrieveAppNameEnv(cliConnection plugin.CliConnection, appName string) ([]string, error) {

	app, err := cliConnection.GetApp(appName)

	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve enviroment for \"%s\", is the app name correct?", appName)
		err = errors.New(msg)
	} else {
		url := fmt.Sprintf("/v2/apps/%s/env", app.Guid)
		output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", url)

		if err != nil {
			msg := fmt.Sprintf("Failed to retrieve enviroment for \"%s\", is the app name correct?", appName)
			err = errors.New(msg)
		}

		return output, err
	}
	return nil, err
}

func (c *JsonQuery) extractCredentialsJSON(envParent string, credKey string, output []string) ([]byte, error) {
	err := errors.New("missing service credentials for application")
	var envJson []byte

	envKey := strings.Join(output, "")
	if strings.Contains(envKey, credKey) {
		var f interface{}
		err = json.Unmarshal([]byte(envKey), &f)
		if err != nil {
			return nil, err
		}

		envJSON := f.(map[string]interface{})
		envParentJSON := envJSON[envParent].(map[string]interface{})
		envJson, err = json.Marshal(envParentJSON[credKey])
		if err != nil {
			return nil, err
		}
	}

	return envJson, err
}

func (c *JsonQuery) exportCredsAsShellVar(credKey string, creds string) {
	vcapServices := fmt.Sprintf("export %s='%s';", credKey, creds)
	fmt.Println(vcapServices)
}

func (c *JsonQuery) exportCredsAsJSON(credKey string, creds string) {
	//vcapServices := fmt.Sprintf("%s", creds)
	vcapServices := fmt.Sprintf("{ \"%s\":%s }", credKey, creds)
	fmt.Println(vcapServices)
}

func (c *JsonQuery) extractAndExportCredentials(envParent string, credKey string, appEnv []string) {
	creds, err := c.extractCredentialsJSON(envParent, credKey, appEnv)
	fatalIf(err)
	//c.exportCredsAsShellVar(credKey, string(creds[:]))
	c.exportCredsAsJSON(credKey, string(creds[:]))
}

// Run plugin start
func (c *JsonQuery) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) > 0 && args[0] == "CLI-MESSAGE-UNINSTALL" {
		return
	}

	appName, err := c.extractAppName(args)
	fatalIf(err)

	appEnv, err := c.retrieveAppNameEnv(cliConnection, appName)
	fatalIf(err)

	c.extractAndExportCredentials("system_env_json", "VCAP_SERVICES", appEnv)
	if len(args) > 2 && args[2] == "--all" {
		//fmt.Println("")
		c.extractAndExportCredentials("application_env_json", "VCAP_APPLICATION", appEnv)
	}
}

// GetMetadata returns plugin metadata
func (c *JsonQuery) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "jsonquery",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "jsonquery",
				Alias:    "jq",
				HelpText: "Export application VCAP_SERVICES to local environment variable.",
				UsageDetails: plugin.Usage{
					Usage: "jsonquery APP_NAME [--all] - Retrieve and export remote application VCAP_SERVICES to local developer environment.",
					Options: map[string]string{
						"all": "Retrieve both VCAP_SERVICES and VCAP_APPLICATION from remote application",
					},
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(JsonQuery))
}
