package statsserver

import (
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"

	peer "gx/ipfs/QmdVrMn1LhB4ybb8hMVaMLXnA8XRSewMnK6YqXKXoTcRvN/go-libp2p-peer"

	swarm ".."
	"../../repo"
)

type server struct {
	router *http.ServeMux
	node   *swarm.Node
}

// var tmpl = template.Must(template.ParseFiles("templates/index.html"))

func Start(listenaddr string, node *swarm.Node) {
	s := &server{
		router: http.NewServeMux(),
		node:   node,
	}

	s.router.HandleFunc("/", s.handleIndex())
	err := http.ListenAndServe(listenaddr, s.router)
	if err != nil {
		panic(err)
	}
}

func (s *server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeState, err := s.node.GetNodeState()
		if err != nil {
			die500(w, err)
			return
		}

		type Peer struct {
			PrettyName string
			Name       string
			Addrs      []string
		}

		type State struct {
			Username       string
			EthAddress     string
			Addrs          []string
			Peers          []Peer
			Repos          []repo.RepoInfo
			ReplicateRepos []string
		}

		var state State

		state.Username = nodeState.User
		state.EthAddress = nodeState.EthAccount
		state.Addrs = nodeState.Addrs
		for _, peerID := range s.node.Host.Peerstore().Peers() {
			p := Peer{PrettyName: peerID.String(), Name: peer.IDB58Encode(peerID)}
			for _, addr := range s.node.Host.Peerstore().Addrs(peerID) {
				p.Addrs = append(p.Addrs, addr.String())
			}
			state.Peers = append(state.Peers, p)
		}
		for _, repo := range nodeState.Repos {
			state.Repos = append(state.Repos, repo)
		}

		state.ReplicateRepos = s.node.Config.Node.ReplicateRepos

		tpl, err := template.New("indexpage").Parse(`
			<html>
			<head>
				<title>Conscience Node stats</title>
				<style>
					body {
						font-family: Consolas, 'Courier New', sans-serif;
					}
				</style>
			</head>
			<body>
				<div>Username: {{ .Username }}</div>
				<div>ETH account: {{ .EthAddress }}</div>
				<div>
					Listen addrs:
					<ul>
						{{ range .Addrs }}
							<li>{{ . }}</li>
						{{ end }}
					</ul>
				</div>
				<div>
					Peers:
					<ul>
						{{ range .Peers }}
							<li>
								<div>{{ .PrettyName }} ({{ .Name }})</div>
								<ul>
									{{ range .Addrs }}
										<li>{{ . }}</li>
									{{ end }}
								</ul>
							</li>
						{{ end }}
					</ul>
				</div>
				<div>
					Replicating repos:
					<ul>
						{{ range .ReplicateRepos }}
							<li>{{ . }}</li>
						{{ end }}
					</ul>
				</div>
			</body>
			</html>
		`)
		if err != nil {
			die500(w, err)
			return
		}

		err = tpl.Execute(w, state)
		if err != nil {
			die500(w, err)
			return
		}
	}
}

func die500(w http.ResponseWriter, err error) {
	log.Errorln("[http]", err)
	w.WriteHeader(500)
	w.Write([]byte("Internal server error: " + err.Error()))
}
