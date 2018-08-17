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

func Start(listenaddr string, node *swarm.Node) {
	s := &server{
		router: http.NewServeMux(),
		node:   node,
	}

	s.router.HandleFunc("/", s.handleIndex())
	s.router.HandleFunc("/set-replication-policy", s.handleAddReplicatedRepo())

	err := http.ListenAndServe(listenaddr, s.router)
	if err != nil {
		panic(err)
	}
}

func (s *server) handleAddReplicatedRepo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			die500(w, err)
			return
		}

		repoID := r.Form.Get("repo")
		shouldReplicate := r.Form.Get("should_replicate") == "true"

		err = s.node.SetReplicationPolicy(repoID, shouldReplicate)
		if err != nil {
			die500(w, err)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (s *server) handleIndex() http.HandlerFunc {
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

	return func(w http.ResponseWriter, r *http.Request) {
		nodeState, err := s.node.GetNodeState()
		if err != nil {
			die500(w, err)
			return
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
				<title>Conscience node stats</title>
				<style>
					body {
						font-family: Consolas, 'Courier New', sans-serif;
						padding: 20px;
					}

					header {
						display: flex;
					}

					header .logo svg {
						width: 80px;
						height: 80px;
					}

					header .text {
						font-family: 'Roboto Condensed', sans-serif;
						font-size: 2.3rem;
					}

					section {
						margin-bottom: 30px;
						border: 1px solid #eaeaea;
						padding: 30px;
						border-radius: 10px;
					}

					label {
						font-weight: bold;
					}
				</style>
			</head>
			<body>
				<header>
					<div class="logo">` + logoSVG + `</div>
					<div class="text">conscience p2p node</div>
				</header>

				<section>
					<div><label>Username:</label> {{ .Username }}</div>
					<div><label>ETH account:</label> {{ .EthAddress }}</div>
					<div>
						<label>Listen addrs:</label>
						<ul>
							{{ range .Addrs }}
								<li>{{ . }}</li>
							{{ end }}
						</ul>
					</div>
				</section>

				<section>
					<label>Peers:</label>
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
				</section>

				<section>
					<label>Replicating repos:</label>
					<ul>
						{{ range .ReplicateRepos }}
							<li>{{ . }}</li>
						{{ end }}
					</ul>

					<div><label>Set replication policy</label></div>
					<form action="/set-replication-policy" method="post">
						<div>Repo ID: <input type="text" name="repo" /></div>
						<div>Should replicate: <input type="checkbox" name="should_replicate" value="true" /></div>
						<div><input type="submit" value="Set" /></div>
					</form>
				</section>
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

var logoSVG = `
	<svg width="200px" height="200px" viewBox="100,100,500,300" xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" xmlns:xlink="http://www.w3.org/1999/xlink">
		<g transform="matrix(3.0755,0,0,3.0755,245.3298,0.6867)">
			<svg width="97" height="101.703" viewBox="2.4989999999999988,0.1479999999999979,97,101.703">
				<defs>
					<linearGradient x1="0" y1="0.5" x2="1" y2="0.5" id="Sj9095p3g3">
						<stop offset="26.54%" stop-color="#000000"></stop>
						<stop offset="39.81%" stop-color="#000000"></stop>
						<stop offset="68.96%" stop-color="#000000"></stop>
						<stop offset="100%" stop-color="#000000"></stop>
					</linearGradient>
				</defs>
				<g>
					<path fill-rule="evenodd" clip-rule="evenodd" d="M82.743,28.662c-5.281,0-9.699-3.506-11.181-8.302l-4.053,0.9l-1.102-4.573  l4.707-1.045c0.636-5.891,5.566-10.494,11.628-10.494c6.494,0,11.757,5.264,11.757,11.757S89.237,28.662,82.743,28.662z   M82.743,9.852c-3.895,0-7.054,3.158-7.054,7.054s3.159,7.054,7.054,7.054s7.054-3.159,7.054-7.054S86.638,9.852,82.743,9.852z   M87.446,75.689c0,11.688-9.475,21.162-21.162,21.162c-11.688,0-21.163-9.475-21.163-21.162c0-5.816,2.349-11.082,6.147-14.907  l-6.562-7.344l3.45-3.196l6.789,7.599c3.281-2.088,7.164-3.313,11.339-3.313C77.972,54.527,87.446,64.001,87.446,75.689z   M66.284,59.229c-9.091,0-16.46,7.369-16.46,16.46s7.369,16.459,16.46,16.459s16.459-7.368,16.459-16.459  S75.375,59.229,66.284,59.229z M51.941,19.902l9.876-2.194l1.103,4.573l-9.876,2.193L51.941,19.902z M40.419,28.662  c0,3.549-1.148,6.814-3.059,9.503l7.661,8.572l-3.45,3.198l-7.491-8.382c-2.799,2.202-6.285,3.569-10.122,3.569  c-9.091,0-16.459-7.369-16.459-16.459s7.369-16.459,16.459-16.459c6.987,0,12.921,4.371,15.309,10.515l8.082-1.796l1.102,4.573  l-8.17,1.815C40.32,27.763,40.419,28.198,40.419,28.662z M23.959,16.871c-6.512,0-11.792,5.279-11.792,11.791  s5.28,11.792,11.792,11.792s11.792-5.28,11.792-11.792S30.472,16.871,23.959,16.871z" fill="url('#Sj9095p3g3')"></path>
				</g>
			</svg>
		</g>
		<g transform="matrix(-1.2592,1.4486,-1.4486,-1.2592,361.7182,132.335)">
			<svg width="97" height="101.703" viewBox="2.4989999999999988,0.1479999999999979,97,101.703">
				<defs>
					<linearGradient x1="0" y1="0.5" x2="1" y2="0.5" id="Sj9095p3gb">
						<stop offset="26.54%" stop-color="#000000"></stop>
						<stop offset="39.81%" stop-color="#000000"></stop>
						<stop offset="68.96%" stop-color="#000000"></stop>
						<stop offset="100%" stop-color="#000000"></stop>
					</linearGradient>
				</defs>
				<g>
					<path fill-rule="evenodd" clip-rule="evenodd" d="M82.743,28.662c-5.281,0-9.699-3.506-11.181-8.302l-4.053,0.9l-1.102-4.573  l4.707-1.045c0.636-5.891,5.566-10.494,11.628-10.494c6.494,0,11.757,5.264,11.757,11.757S89.237,28.662,82.743,28.662z   M82.743,9.852c-3.895,0-7.054,3.158-7.054,7.054s3.159,7.054,7.054,7.054s7.054-3.159,7.054-7.054S86.638,9.852,82.743,9.852z   M87.446,75.689c0,11.688-9.475,21.162-21.162,21.162c-11.688,0-21.163-9.475-21.163-21.162c0-5.816,2.349-11.082,6.147-14.907  l-6.562-7.344l3.45-3.196l6.789,7.599c3.281-2.088,7.164-3.313,11.339-3.313C77.972,54.527,87.446,64.001,87.446,75.689z   M66.284,59.229c-9.091,0-16.46,7.369-16.46,16.46s7.369,16.459,16.46,16.459s16.459-7.368,16.459-16.459  S75.375,59.229,66.284,59.229z M51.941,19.902l9.876-2.194l1.103,4.573l-9.876,2.193L51.941,19.902z M40.419,28.662  c0,3.549-1.148,6.814-3.059,9.503l7.661,8.572l-3.45,3.198l-7.491-8.382c-2.799,2.202-6.285,3.569-10.122,3.569  c-9.091,0-16.459-7.369-16.459-16.459s7.369-16.459,16.459-16.459c6.987,0,12.921,4.371,15.309,10.515l8.082-1.796l1.102,4.573  l-8.17,1.815C40.32,27.763,40.419,28.198,40.419,28.662z M23.959,16.871c-6.512,0-11.792,5.279-11.792,11.791  s5.28,11.792,11.792,11.792s11.792-5.28,11.792-11.792S30.472,16.871,23.959,16.871z" fill="url('#Sj9095p3gb')"></path>
				</g>
			</svg>
		</g>
		<g transform="matrix(1.0277,0,0,1.0277,417.5046,39.2911)">
			<svg width="630" height="188" style="overflow: visible;">
				<defs>
					<linearGradient x1="0" y1="94" x2="630" y2="94" gradientUnits="userSpaceOnUse" id="Sj9095p3gl">
						<stop offset="0%" stop-color="#000000"></stop>
						<stop offset="100%" stop-color="#000000"></stop>
					</linearGradient>
				</defs>
				<path fill="none" d="M-50,168 C295.15625,167 335.15625,167 680.3125,168" style=""></path>
			</svg>
		</g>
	</svg>
`
