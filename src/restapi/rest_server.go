package restapi

import (
	"net/http"

	"github.com/RiV-chain/RiV-mesh/src/restapi"
	"github.com/RiV-chain/RiVPN/src/config"
)

type RestServer struct {
	server *restapi.RestServer
	config *config.NodeConfig
}

func NewRestServer(server *restapi.RestServer, cfg *config.NodeConfig) (*restapi.RestServer, error) {
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

func (a *RestServer) Serve() error {
	return a.server.Serve()
}
