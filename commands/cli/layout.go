package main

import (
	"log"

	"os"

	"path/filepath"

	"fmt"

	"context"
	"encoding/json"
	"io/ioutil"

	"strings"

	"github.com/tsocial/tessellate/server"
	"gopkg.in/alecthomas/kingpin.v2"
)

type layout struct {
	id          string
	workspaceId string
	dirName     string
}

func (cm *layout) layoutCreate(c *kingpin.ParseContext) error {
	if _, err := os.Stat(cm.dirName); err != nil {
		log.Printf("Directory '%s' does not exist\n", cm.dirName)
	}

	var files []string

	if err := filepath.Walk(cm.dirName, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" || strings.Contains(path, "tfvars") {
			log.Printf("skipping %s", path)
			return nil
		}

		files = append(files, path)
		return nil

	}); err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no json file found in directory %s", cm.dirName)
	}

	var maps []interface{}

	for _, f := range files {
		log.Println(f)
		fBytes, err := ioutil.ReadFile(f)
		if err != nil {
			log.Println(err)
			return err
		}

		var fObj interface{}
		if err := json.Unmarshal(fBytes, &fObj); err != nil {
			log.Printf("invald json file: %s", f)
			return err
		}

		maps = append(maps, fObj)
	}

	log.Println(maps)

	finalMap := mergeMaps(maps...)

	prettyPrint(finalMap)
	layoutBytes, err := json.Marshal(finalMap)
	if err != nil {
		log.Println(err)
		return err
	}

	req := &server.SaveLayoutRequest{
		Id:          cm.id,
		WorkspaceId: cm.workspaceId,
		Plan:        layoutBytes,
	}

	_, err = getClient().SaveLayout(context.Background(), req)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (cm *layout) layoutGet(c *kingpin.ParseContext) error {
	req := &server.LayoutRequest{
		Id:          cm.id,
		WorkspaceId: cm.workspaceId,
	}

	resp, err := getClient().GetLayout(context.Background(), req)
	if err != nil {
		return err
	}

	var plan interface{}
	if err := json.Unmarshal(resp.Plan, &plan); err != nil {
		return err
	}

	prettyPrint(plan)
	return nil
}

func addLayoutCommands(app *kingpin.Application) {
	lCLI := app.Command("layout", "Commands for layout")

	clm := &layout{}
	cl := lCLI.Command("create", "Create Layout").Action(clm.layoutCreate)

	cl.Flag("id", "Name of the layout").Required().StringVar(&clm.id)
	cl.Flag("workspace-id", "Workspace name").Required().StringVar(&clm.workspaceId)
	cl.Flag("dir", "Absolute path of directory where layout files exist").Required().StringVar(&clm.dirName)

	gl := lCLI.Command("get", "Get Layout").Action(clm.layoutGet)

	gl.Flag("id", "Name of the layout").Required().StringVar(&clm.id)
	gl.Flag("workspace-id", "Workspace name").Required().StringVar(&clm.workspaceId)
}

func mergeMaps(maps ...interface{}) interface{} {
	if len(maps) == 1 {
		return maps[0]
	}

	if len(maps) == 2 {
		return merge(maps[0], maps[1])
	}

	merged := merge(maps[0], maps[1])

	log.Println("-------------------------")
	prettyPrint(merged)
	log.Println("-------------------------")
	return mergeMaps(merged, maps[2:])
}

func merge(x1, x2 interface{}) interface{} {
	switch x1 := x1.(type) {
	case map[string]interface{}:
		x2, ok := x2.(map[string]interface{})
		if !ok {
			return x1
		}
		for k, v2 := range x2 {
			if v1, ok := x1[k]; ok {
				x1[k] = merge(v1, v2)
			} else {
				x1[k] = v2
			}
		}
	case nil:
		// merge(nil, map[string]interface{...}) -> map[string]interface{...}
		x2, ok := x2.(map[string]interface{})
		if ok {
			return x2
		}
	}
	return x1
}
