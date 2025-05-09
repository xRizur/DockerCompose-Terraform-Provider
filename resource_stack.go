package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type ComposeService struct {
	Name                string
	Image               string
	Restart             string
	Ports               []string
	DependsOn           []string
	Environment         map[string]string
	Command             []string
	Entrypoint          []string
	Replicas            int
	HealthcheckTest     string
	HealthcheckInterval string
	HealthcheckRetries  int
	ExtraConfig         map[string]interface{}
}

type ComposeNetwork struct {
	Name   string
	Driver string
}

type ComposeVolume struct {
	Name   string
	Driver string
}

const composeTemplate = `
services:
{{- range .Services }}
  {{ .Name }}:
    image: "{{ .Image }}"
    restart: "{{ .Restart }}"
    {{- if .Ports }}
    ports:
      {{- range .Ports }}
      - "{{ . }}"
      {{- end }}
    {{- end }}
    {{- if .DependsOn }}
    depends_on:
      {{- range .DependsOn }}
      - "{{ . }}"
      {{- end }}
    {{- end }}
    {{- if .Environment }}
    environment:
      {{- range $key, $value := .Environment }}
      - "{{ $key }}={{ $value }}"
      {{- end }}
    {{- end }}
    {{- if .Command }}
    command: {{ .Command }}
    {{- end }}
    {{- if .Entrypoint }}
    entrypoint: {{ .Entrypoint }}
    {{- end }}
    deploy:
      replicas: {{ .Replicas }}
    {{- if and (ne .HealthcheckTest "") (ne .HealthcheckInterval "") }}
    healthcheck:
      test: ["CMD", "{{ .HealthcheckTest }}"]
      interval: "{{ .HealthcheckInterval }}"
      retries: {{ .HealthcheckRetries }}
    {{- end }}
    {{- if .ExtraConfig }}
    {{ .ExtraConfig | toYaml | indent 4 }}
    {{- end }}
{{- end }}

{{- if gt (len .Networks) 0 }}
networks:
{{- range .Networks }}
  {{ .Name }}:
    driver: "{{ .Driver }}"
{{- end }}
{{- end }}

{{- if gt (len .Volumes) 0 }}
volumes:
{{- range .Volumes }}
  {{ .Name }}:
    driver: "{{ .Driver }}"
{{- end }}
{{- end }}
`

func resourceComposeStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceComposeCreate,
		Read:   resourceComposeRead,
		Update: resourceComposeUpdate,
		Delete: resourceComposeDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"service": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name":                 {Type: schema.TypeString, Required: true},
						"image":                {Type: schema.TypeString, Required: true},
						"restart":              {Type: schema.TypeString, Optional: true, Default: "always"},
						"ports":                {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"depends_on":           {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"environment":          {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"command":              {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"entrypoint":           {Type: schema.TypeList, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"replicas":             {Type: schema.TypeInt, Optional: true, Default: 1},
						"healthcheck_test":     {Type: schema.TypeString, Optional: true},
						"healthcheck_interval": {Type: schema.TypeString, Optional: true},
						"healthcheck_retries":  {Type: schema.TypeInt, Optional: true},
						"extra_config":         {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
					},
				},
			},
			"network": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name":   {Type: schema.TypeString, Required: true},
						"driver": {Type: schema.TypeString, Optional: true, Default: "bridge"},
					},
				},
			},
			"volume": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name":   {Type: schema.TypeString, Required: true},
						"driver": {Type: schema.TypeString, Optional: true, Default: "local"},
					},
				},
			},
		},
	}
}

func resourceComposeCreate(d *schema.ResourceData, m interface{}) error {
	stackName := d.Get("name").(string)

	services, networks, volumes := parseServicesAndNetworks(d)

	if err := generateComposeFile("docker-compose.yml", services, networks, volumes); err != nil {
		return err
	}

	cmd := exec.Command("docker", "compose", "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running docker compose: %s", string(output))
	}

	d.SetId(stackName)
	return nil
}

func resourceComposeDelete(d *schema.ResourceData, m interface{}) error {
	cmd := exec.Command("docker", "compose", "down")
	_, err := cmd.CombinedOutput()
	return err
}

func parseServicesAndNetworks(d *schema.ResourceData) ([]ComposeService, []ComposeNetwork, []ComposeVolume) {
	rawServices := d.Get("service").(*schema.Set).List()
	rawNetworks := d.Get("network").(*schema.Set).List()
	rawVolumes := d.Get("volume").(*schema.Set).List()

	var services []ComposeService
	for _, raw := range rawServices {
		svcMap := raw.(map[string]interface{})
		service := ComposeService{
			Name:                svcMap["name"].(string),
			Image:               svcMap["image"].(string),
			Restart:             getString(svcMap, "restart", "always"),
			Ports:               getStringList(svcMap, "ports"),
			DependsOn:           getStringList(svcMap, "depends_on"),
			Environment:         getStringMap(svcMap, "environment"),
			Command:             getStringList(svcMap, "command"),
			Entrypoint:          getStringList(svcMap, "entrypoint"),
			Replicas:            getInt(svcMap, "replicas", 1),
			HealthcheckTest:     getString(svcMap, "healthcheck_test", "curl -f http://localhost"),
			HealthcheckInterval: getString(svcMap, "healthcheck_interval", "30s"),
			HealthcheckRetries:  getInt(svcMap, "healthcheck_retries", 3),
			ExtraConfig:         getMap(svcMap, "extra_config"),
		}
		services = append(services, service)
	}

	var networks []ComposeNetwork
	for _, raw := range rawNetworks {
		netMap := raw.(map[string]interface{})
		networks = append(networks, ComposeNetwork{
			Name:   netMap["name"].(string),
			Driver: getString(netMap, "driver", "bridge"),
		})
	}

	var volumes []ComposeVolume
	for _, raw := range rawVolumes {
		volMap := raw.(map[string]interface{})
		volumes = append(volumes, ComposeVolume{
			Name:   volMap["name"].(string),
			Driver: getString(volMap, "driver", "local"),
		})
	}

	return services, networks, volumes
}

func resourceComposeRead(d *schema.ResourceData, m interface{}) error {
	// Sprawdź, czy plik docker-compose.yml istnieje
	if _, err := os.Stat("docker-compose.yml"); os.IsNotExist(err) {
		fmt.Println("docker-compose.yml not found, regenerating from state...")
		if err := resourceComposeCreate(d, m); err != nil {
			return fmt.Errorf("error regenerating docker-compose file: %s", err)
		}
	}

	// Teraz wywołaj "docker compose ps" by sprawdzić, czy kontenery działają
	cmd := exec.Command("docker", "compose", "ps", "--services")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error checking docker-compose state: %s", string(output))
	}

	outputStr := string(bytes.TrimSpace(output))
	if outputStr == "" {
		// Jeśli kontenery nie działają, oznacz zasób jako usunięty
		d.SetId("")
		fmt.Println("No running services found. Marking resource as destroyed.")
		return nil
	}

	fmt.Println("Terraform state matches running containers:", outputStr)
	return nil
}

func resourceComposeUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceComposeCreate(d, m)
}

func generateComposeFile(filename string, services []ComposeService, networks []ComposeNetwork, volumes []ComposeVolume) error {
	var tpl bytes.Buffer
	tmpl, err := template.New("compose").Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
		"toYaml": myToYaml,
	}).Parse(composeTemplate)
	if err != nil {
		return err
	}

	err = tmpl.Execute(&tpl, map[string]interface{}{
		"Services": services,
		"Networks": networks,
		"Volumes":  volumes,
	})
	if err != nil {
		return err
	}

	finalOutput := bytes.TrimSpace(tpl.Bytes())
	return os.WriteFile(filename, finalOutput, 0644)
}
