package restapi

import (
	"encoding/json"
	"net/http"

	d "github.com/RiV-chain/RiV-mesh/src/defaults"
	"github.com/RiV-chain/RiV-mesh/src/restapi"
	c "github.com/RiV-chain/RiVPN/src/config"
	"github.com/RiV-chain/RiVPN/src/defaults"
)

type RestServer struct {
	server *restapi.RestServer
	config *c.NodeConfig
}

func NewRestServer(server *restapi.RestServer, cfg *c.NodeConfig) (*restapi.RestServer, error) {
	a := &RestServer{
		server,
		cfg,
	}
	//add CKR for REST handlers here
	err := a.server.AddHandler(restapi.ApiHandler{
		Method: "GET", Pattern: "/api/tunnelrouting", Desc: "Show TunnelRouting settings", Handler: a.getApiTunnelRouting,
	})
	return a.server, err
}

// @Summary		Show TunnelRouting settings.
// @Produce		json
// @Success		200		{string}	string		"ok"
// @Failure		400		{error}		error		"Method not allowed"
// @Failure		401		{error}		error		"Authentication failed"
// @Router		/tunnelrouting [get]
func (a *RestServer) getApiTunnelRouting(w http.ResponseWriter, r *http.Request) {
	restapi.WriteJson(w, r, a.config.TunnelRoutingConfig)
}

// @Summary		Set TunnelRouting settings.
// @Produce		json
// @Success		204		{string}	string		"No content"
// @Failure		400		{error}		error		"Bad request"
// @Failure		401		{error}		error		"Authentication failed"
// @Failure		500		{error}		error		"Internal error"
// @Router		/tunnelrouting [put]
func (a *RestServer) putApiTunnelRouting(w http.ResponseWriter, r *http.Request) {
	var tunnelRouting c.TunnelRoutingConfig
	err := json.NewDecoder(r.Body).Decode(&tunnelRouting)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	a.saveConfig(func(cfg *c.NodeConfig) {
		cfg.TunnelRoutingConfig = tunnelRouting
	}, r)
}

func (a *RestServer) saveConfig(setConfigFields func(*c.NodeConfig), r *http.Request) {
	if len(a.server.ConfigFn) > 0 {
		saveHeaders := r.Header["Riv-Save-Config"]
		if len(saveHeaders) > 0 && saveHeaders[0] == "true" {
			cfg, err := d.ReadConfig(a.server.ConfigFn)
			config := &c.NodeConfig{
				NodeConfig: cfg,
			}
			if err == nil {
				if setConfigFields != nil {
					setConfigFields(config)
				}
				err := defaults.WriteConfig(a.server.ConfigFn, config)
				if err != nil {
					a.server.Log.Errorln("Config file write error:", err)
				}
			} else {
				a.server.Log.Errorln("Config file read error:", err)
			}
		}
	}
}

func (a *RestServer) Serve() error {
	return a.server.Serve()
}
