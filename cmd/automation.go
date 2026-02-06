package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optremove"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type colorUp struct{}

type colorDestroy struct{}

type parallelismDown struct{}

type parallelismUp struct{}

type forceRemove struct{}

type MicroStack struct {
	FullName string
	Name     string
	Path     string
	PreRun   []string
	PostRun  []string
}

type StackMap map[int]*MicroStack

func (colorUp) ApplyOption(opts *optup.Options) {
	opts.Color = "always"
}

func (colorDestroy) ApplyOption(opts *optdestroy.Options) {
	opts.Color = "always"
}

func (parallelismDown) ApplyOption(opts *optdestroy.Options) {
	opts.Parallel = 4
}

func (parallelismUp) ApplyOption(opts *optup.Options) {
	opts.Parallel = 4
}

func (forceRemove) ApplyOption(opts *optremove.Options) {
	opts.Force = true
}

type PrePostCmd func(ctx context.Context, s auto.Stack)

var PrePostFn = map[string]PrePostCmd{
	"cleanupLke":        cleanupLke,
	"deleteObj":         deleteObj,
	"addNodeBalancerId": addNodeBalancerId,
	"rmNodeBalancerId":  rmNodeBalancerId,
}

func (stk *MicroStack) GetFullName(ctx context.Context) {
	// use microstack name value from Pulumi.yaml for the FQSN
	var data map[string]any

	file := filepath.Join(stk.Path, "Pulumi.yaml")

	f, err := os.ReadFile(file)
	if err != nil {
		logger.Error("get fully qualified stack name: " + err.Error())
	}

	if err = yaml.Unmarshal(f, &data); err != nil {
		logger.Error("yaml unmarshal microstack data: " + err.Error())
	}

	n, _ := isValid(data["name"])

	org := viper.GetString("pulumiOrg")
	stk.FullName = filepath.Join(org, n, platform.Stack) // format: org/projName/stack
}

func (stk *MicroStack) PrePostRun(ctx context.Context, s auto.Stack, action string) {
	doit := func(fn string) {
		if _, ok := PrePostFn[fn]; ok {
			PrePostFn[fn](ctx, s)
		}
	}

	switch action {
	case "pre": //nolint:goconst
		for _, i := range stk.PreRun {
			msg := fmt.Sprintf("%s stack PreRun func: %v", stk.Name, i)
			logger.Info(msg)
			doit(i)
		}
	case "post":
		for _, i := range stk.PostRun {
			msg := fmt.Sprintf("%s stack PostRun func: %v", stk.Name, i)
			logger.Info(msg)
			doit(i)
		}
	}
}

func (stk *MicroStack) Up(ctx context.Context) {
	stdout := optup.ProgressStreams(os.Stdout)
	s := initLocalStack(ctx, stk)

	if _, err := s.Refresh(ctx); err != nil {
		logger.Error("failed to refresh stack on pulumi deploy: " + err.Error())
	}

	stk.PrePostRun(ctx, s, "pre")
	msg := fmt.Sprintf("deploying %s stack", stk.Name)
	logger.Info(msg)

	_, err := s.Up(ctx, stdout, colorUp{}, parallelismUp{})
	if err != nil {
		logger.Error("failed to deploy stack: " + err.Error())
	}

	stk.PrePostRun(ctx, s, "post")
}

func (stk *MicroStack) Down(ctx context.Context) *MicroStack {
	if ok := stackExists(ctx, stk.FullName); !ok {
		return nil
	}

	stdout := optdestroy.ProgressStreams(os.Stdout)
	s := initLocalStack(ctx, stk)

	if _, err := s.Refresh(ctx); err != nil {
		logger.Error("failed to refresh stack on pulumi destroy: " + err.Error())
	}

	stk.PrePostRun(ctx, s, "pre")
	msg := fmt.Sprintf("destoying %s stack", stk.Name)
	logger.Info(msg)

	_, err := s.Destroy(ctx, stdout, colorDestroy{}, parallelismDown{})
	if err != nil {
		logger.Error("failed to destroy stack: " + err.Error())
	}

	stk.PrePostRun(ctx, s, "post")

	return stk
}

func (stk *MicroStack) Remove(ctx context.Context) {
	s := initLocalStack(ctx, stk)
	ws := s.Workspace()

	msg := fmt.Sprintf("purging %s stack", stk.Name)
	logger.Info(msg)

	if err := ws.RemoveStack(ctx, stk.FullName, forceRemove{}); err != nil {
		logger.Error("failed to remove stack: " + err.Error())
	}
}

func stackExists(ctx context.Context, fqsn string) bool {
	auth := "token %" + os.Getenv("PULUMI_ACCESS_TOKEN")
	apiURL := `https://api.pulumi.com/api/stacks/` + fqsn

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		logger.Error("create new http request: " + err.Error())
	}

	req.Header.Add("Accept", "application/vnd.pulumi+8")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", auth)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("client http request: " + err.Error())
	}

	defer res.Body.Close()

	return res.StatusCode != http.StatusNotFound
}

func initLocalStack(ctx context.Context, stk *MicroStack) auto.Stack {
	s, err := auto.UpsertStackLocalSource(ctx, stk.FullName, stk.Path)
	if err != nil {
		msg := fmt.Sprintf("failed to get %s stack: %v", stk.Name, err.Error())
		logger.Error(msg)
	}

	msg := fmt.Sprintf("using %s stack", stk.Name)
	logger.Info(msg)

	return s
}

func addNodeBalancerId(ctx context.Context, s auto.Stack) {
	nbid, _ := getResourceVar(ctx, "loadbalancerId", s)

	err := s.SetConfig(ctx, "nodebalancer-id", auto.ConfigValue{Value: nbid})
	if err != nil {
		logger.Error("failed to set loadbalancerId in pulumi stack config")
	}
}

func rmNodeBalancerId(ctx context.Context, s auto.Stack) {
	_, lkeId := getResourceVar(ctx, "lkeId", s)

	deleteNodeBalancers(ctx, lkeId)

	if err := s.RemoveConfig(ctx, "nodebalancer-id"); err != nil {
		logger.Error("failed to remove nodebalancer-id from pulumi stack config")
	}
}

func cleanupLke(ctx context.Context, s auto.Stack) {
	_, lkeId := getResourceVar(ctx, "lkeId", s)

	purgeLkeClusterResources(ctx, lkeId)
}

func deleteObj(ctx context.Context, s auto.Stack) {
	objRemote := s3Remote{
		Endpoint:     platform.Region + "-1.linodeobjects.com",
		Remote:       platform.Name,
		PurgeEnabled: false,
	}

	buckets, err := s.GetConfig(ctx, "apl:objBuckets")
	if err != nil {
		logger.Error("load obj buckets from esc (auto api): " + err.Error())
		os.Exit(1)
	}

	if err := json.Unmarshal([]byte(buckets.Value), &objRemote.Buckets); err != nil {
		logger.Error("json unmarshal obj bucket data: " + err.Error())
	}

	key, err := s.GetConfig(ctx, "apl:objKey")
	if err != nil {
		logger.Error("load obj keys from esc (auto api): " + err.Error())
	}

	if err := json.Unmarshal([]byte(key.Value), &objRemote); err != nil {
		logger.Error("json unmarshal obj key data: " + err.Error())
	}

	objRemote.Init(ctx)
	objRemote.Purge(ctx)
}

func getResourceVar(ctx context.Context, v string, s auto.Stack) (string, int) {
	switch v {
	case "lkeId":
		idVal, err := s.GetConfig(ctx, "lkeId")
		if err != nil {
			if strings.Contains(err.Error(), "unable to read config: exit status 255") {
				// If not in stack's state, try to get it from esc environment,
				// in case this is just an issue of ordering.
				orgName := viper.GetString("pulumiOrg")
				esc := NewEnvObject(orgName, platform.Name, platform.Stack)
				_, id := esc.GetConfig("lkeId") //nolint:contextcheck

				return "", id
			}

			logger.Error("get lkeId: " + err.Error())
		}

		id, err := strconv.Atoi(idVal.Value)
		if err != nil {
			logger.Error(err.Error())
		}

		return "", id
	case "loadbalancerId":
		res, err := s.Outputs(ctx)
		if err != nil {
			logger.Error(err.Error())
		}

		_, output := isValid(res["infraStackOutputs"].Value)

		nbid, ok := output["loadbalancerId"]
		if !ok {
			logger.Error("unable to find loadbalancerId in output")
		}
		// type check the final value before return
		n, _ := isValid(nbid)

		return n, 0
	}

	return "", 0
}

func isValid(i any) (string, map[string]any) {
	switch v := i.(type) {
	case string:
		if v == "" {
			logger.Error("string type variable has zero value")
		}

		return v, nil
	case map[string]any:
		if len(v) == 0 {
			logger.Error("map[string]any type variable has zero value")
		}

		return "", v
	default:
		logger.Error("type assertion failed")
	}

	return "", nil
}
