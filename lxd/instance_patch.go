package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/canonical/lxd/lxd/db"
	deviceConfig "github.com/canonical/lxd/lxd/device/config"
	"github.com/canonical/lxd/lxd/instance"
	projecthelpers "github.com/canonical/lxd/lxd/project"
	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/osarch"
)

// swagger:operation PATCH /1.0/instances/{name} instances instance_patch
//
// Partially update the instance
//
// Updates a subset of the instance configuration
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
//   - in: query
//     name: project
//     description: Project name
//     type: string
//     example: default
//   - in: body
//     name: instance
//     description: Update request
//     schema:
//       $ref: "#/definitions/InstancePut"
// responses:
//   "200":
//     $ref: "#/responses/EmptySyncResponse"
//   "400":
//     $ref: "#/responses/BadRequest"
//   "403":
//     $ref: "#/responses/Forbidden"
//   "500":
//     $ref: "#/responses/InternalServerError"
func instancePatch(d *Daemon, r *http.Request) response.Response {
	instanceType, err := urlInstanceTypeDetect(r)
	if err != nil {
		return response.SmartError(err)
	}

	projectName := projectParam(r)

	// Get the container
	name := mux.Vars(r)["name"]

	// Handle requests targeted to a container on a different node
	resp, err := forwardedResponseIfInstanceIsRemote(d, r, projectName, name, instanceType)
	if err != nil {
		return response.SmartError(err)
	}
	if resp != nil {
		return resp
	}

	c, err := instance.LoadByProjectAndName(d.State(), projectName, name)
	if err != nil {
		return response.NotFound(err)
	}

	// Validate the ETag
	etag := []interface{}{c.Architecture(), c.LocalConfig(), c.LocalDevices(), c.IsEphemeral(), c.Profiles()}
	err = util.EtagCheck(r, etag)
	if err != nil {
		return response.PreconditionFailed(err)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return response.InternalError(err)
	}

	rdr1 := ioutil.NopCloser(bytes.NewBuffer(body))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(body))

	reqRaw := shared.Jmap{}
	if err := json.NewDecoder(rdr1).Decode(&reqRaw); err != nil {
		return response.BadRequest(err)
	}

	req := api.InstancePut{}
	if err := json.NewDecoder(rdr2).Decode(&req); err != nil {
		return response.BadRequest(err)
	}

	if req.Restore != "" {
		return response.BadRequest(fmt.Errorf("Can't call PATCH in restore mode"))
	}

	// Check if architecture was passed
	var architecture int
	_, err = reqRaw.GetString("architecture")
	if err != nil {
		architecture = c.Architecture()
	} else {
		architecture, err = osarch.ArchitectureId(req.Architecture)
		if err != nil {
			architecture = 0
		}
	}

	// Check if ephemeral was passed
	_, err = reqRaw.GetBool("ephemeral")
	if err != nil {
		req.Ephemeral = c.IsEphemeral()
	}

	// Check if profiles was passed
	if req.Profiles == nil {
		req.Profiles = c.Profiles()
	}

	// Check if config was passed
	if req.Config == nil {
		req.Config = c.LocalConfig()
	} else {
		for k, v := range c.LocalConfig() {
			_, ok := req.Config[k]
			if !ok {
				req.Config[k] = v
			}
		}
	}

	// Check if devices was passed
	if req.Devices == nil {
		req.Devices = c.LocalDevices().CloneNative()
	} else {
		for k, v := range c.LocalDevices() {
			_, ok := req.Devices[k]
			if !ok {
				req.Devices[k] = v
			}
		}
	}

	// Check project limits.
	err = d.cluster.Transaction(func(tx *db.ClusterTx) error {
		return projecthelpers.AllowInstanceUpdate(tx, projectName, name, req, c.LocalConfig())
	})
	if err != nil {
		return response.SmartError(err)
	}

	// Update container configuration
	args := db.InstanceArgs{
		Architecture: architecture,
		Config:       req.Config,
		Description:  req.Description,
		Devices:      deviceConfig.NewDevices(req.Devices),
		Ephemeral:    req.Ephemeral,
		Profiles:     req.Profiles,
		Project:      projectName,
	}

	err = c.Update(args, true)
	if err != nil {
		return response.SmartError(err)
	}

	return response.EmptySyncResponse
}
