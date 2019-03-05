package nodehttp

import (
	"context"
	"expvar"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/brynbellomy/debugcharts"
	git "github.com/libgit2/git2go"

	peer "github.com/libp2p/go-libp2p-peer"

	"github.com/Conscience/protocol/log"
	"github.com/Conscience/protocol/repo"
	"github.com/Conscience/protocol/swarm"
	"github.com/Conscience/protocol/swarm/logger"
	"github.com/Conscience/protocol/util"
)

type Server struct {
	server *http.Server
	node   *swarm.Node
}

func New(node *swarm.Node) *Server {
	s := &Server{
		node: node,
	}

	router := http.NewServeMux()
	router.HandleFunc("/", s.handleIndex())
	router.HandleFunc("/set-replication-policy", s.handleAddReplicatedRepo())
	router.HandleFunc("/remove-peer", s.handleRemovePeer())
	router.HandleFunc("/untrack-repo", s.handleUntrackRepo())
	router.Handle("/debug/vars", expvar.Handler())
	debugcharts.RegisterHandlers(router)

	username := node.Config.Node.HTTPUsername
	password := node.Config.Node.HTTPPassword
	handler := BasicAuth(username, password, router)

	s.server = &http.Server{Addr: node.Config.Node.HTTPListenAddr, Handler: handler}

	return s
}

func (s *Server) Start() {
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) handleRemovePeer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			die500(w, err)
			return
		}

		peerIDStr := r.Form.Get("peerid")
		if peerIDStr == "" {
			die500(w, err)
			return
		}

		peerID, err := peer.IDB58Decode(peerIDStr)
		if err != nil {
			die500(w, err)
			return
		}

		err = s.node.RemovePeer(peerID)
		if err != nil {
			die500(w, err)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (s *Server) handleUntrackRepo() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			die500(w, err)
			return
		}

		repoPath := req.Form.Get("repoPath")
		if repoPath == "" {
			die500(w, err)
			return
		}

		r, err := s.node.RepoManager().RepoAtPathOrID(repoPath, "")
		if err != nil {
			die500(w, err)
			return
		}

		err = s.node.RepoManager().UntrackRepo(repoPath)
		if err != nil {
			die500(w, err)
			return
		}

		repoID, err := r.RepoID()
		if err != nil {
			die500(w, err)
			return
		}

		err = s.node.SetReplicationPolicy(repoID, false)
		if err != nil {
			die500(w, err)
			return
		}

		http.Redirect(w, req, "/", http.StatusFound)
	}
}

func (s *Server) handleAddReplicatedRepo() http.HandlerFunc {
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

func (s *Server) handleIndex() http.HandlerFunc {
	type Peer struct {
		PrettyName string
		Name       string
		Addrs      []string
	}

	type RefMapping struct {
		RefName      string
		LocalCommit  string
		RemoteCommit string
	}

	type RepoInfo struct {
		RepoID string
		Path   string
		Refs   []RefMapping
	}

	type EnvVar struct {
		Name  string
		Value string
	}

	type State struct {
		Username             string
		EthAddress           string
		ProtocolContractAddr string
		RPCListenAddr        string
		Addrs                []string
		Peers                []Peer
		PeersConnected       int
		LocalRepos           []RepoInfo
		ReplicateRepos       []string
		Logs                 []logger.Entry
		Env                  []EnvVar
		GlobalConnStats      struct {
			TotalIn  string
			TotalOut string
			RateIn   string
			RateOut  string
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		addrs := make([]string, 0)
		for _, addr := range s.node.Addrs() {
			addrs = append(addrs, fmt.Sprintf("%v/p2p/%v", addr.String(), s.node.ID().Pretty()))
		}

		var username string
		{
			var err error
			ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
			username, err = s.node.GetUsername(ctx)
			if err != nil {
				username = "<error fetching username>"
			}
		}

		var state State

		state.Username = username
		state.EthAddress = s.node.EthAddress().Hex()
		state.ProtocolContractAddr = s.node.Config.Node.ProtocolContract
		state.RPCListenAddr = s.node.Config.Node.RPCListenNetwork + ":" + s.node.Config.Node.RPCListenHost
		state.Addrs = addrs

		for _, pinfo := range s.node.Peers() {
			p := Peer{PrettyName: pinfo.ID.String(), Name: peer.IDB58Encode(pinfo.ID)}
			for _, addr := range pinfo.Addrs {
				p.Addrs = append(p.Addrs, addr.String())
			}
			state.Peers = append(state.Peers, p)
		}

		state.PeersConnected = len(s.node.Conns())

		err := s.node.RepoManager().ForEachRepo(func(r *repo.Repo) error {
			repoID, err := r.RepoID()
			if err != nil {
				return err
			}

			rIter, err := r.NewReferenceIterator()
			if err != nil {
				return err
			}
			defer rIter.Free()

			remoteRefs, _, err := s.node.GetRemoteRefs(context.TODO(), repoID, 50, 0)
			if err != nil {
				return err
			}

			refmap := map[string]RefMapping{}
			for {
				ref, err := rIter.Next()
				if git.IsErrorCode(err, git.ErrIterOver) {
					break
				} else if err != nil {
					return err
				}

				ref, err = ref.Resolve()
				if err != nil {
					return err
				}

				refname := ref.Name()

				refmap[refname] = RefMapping{
					RefName:      refname,
					LocalCommit:  ref.Target().String(),
					RemoteCommit: remoteRefs[refname].CommitHash,
				}
			}

			// Overlay any remote refs that don't exist locally
			for refname := range remoteRefs {
				if _, exists := refmap[refname]; !exists {
					refmap[refname] = RefMapping{
						RefName:      refname,
						RemoteCommit: remoteRefs[refname].CommitHash,
					}
				}
			}
			refs := []RefMapping{}
			for _, x := range refmap {
				refs = append(refs, x)
			}
			sort.Slice(refs, func(i, j int) bool { return refs[i].RefName < refs[j].RefName })

			state.LocalRepos = append(state.LocalRepos, RepoInfo{
				RepoID: repoID,
				Path:   r.Path(),
				Refs:   refs,
			})
			return nil
		})
		if err != nil {
			die500(w, err)
			return
		}

		state.ReplicateRepos = s.node.Config.Node.ReplicateRepos

		state.Logs = logger.GetLogs()

		for _, x := range os.Environ() {
			parts := strings.Split(x, "=")
			state.Env = append(state.Env, EnvVar{parts[0], parts[1]})
		}

		stats := s.node.GetBandwidthTotals()
		state.GlobalConnStats.TotalIn = util.HumanizeBytes(float64(stats.TotalIn))
		state.GlobalConnStats.TotalOut = util.HumanizeBytes(float64(stats.TotalOut))
		state.GlobalConnStats.RateIn = util.HumanizeBytes(stats.RateIn)
		state.GlobalConnStats.RateOut = util.HumanizeBytes(stats.RateOut)

		err = tplIndex.Execute(w, state)
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

var tplIndex = template.Must(template.New("indexpage").Parse(`
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

            section.section-peers ul li form {
                display: inline;
            }

            section.section-environment ul li {
                word-break: break-all;
            }

            section.section-environment ul li .value {
                color: #9c9c9c;
                font-size: 0.9em;
            }

            .log.error {
                color: red;
            }

            .log.warning {
                color: orange;
            }

            .log.info {
                color: black;
            }

            .log.debug {
                color: grey;
            }

            .local-repos table {
                padding: 10px;
            }

            .local-repos table thead {
                font-weight: bold;
                text-decoration: underline;
            }

            .local-repos table td {
                padding: 0 8px;
            }

            .local-repos .toggle-refs {
                cursor: pointer;
                color: blue;
            }

            .hidden {
                display: none;
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
            <div><label>Protocol contract:</label> {{ .ProtocolContractAddr }}</div>
            <div><label>RPC listen addr:</label> {{ .RPCListenAddr }}</div>
            <div>
                <label>Listen addrs:</label>
                <ul>
                    {{ range .Addrs }}
                        <li>{{ . }}</li>
                    {{ end }}
                </ul>
            </div>
        </section>

        <section class="section-peers">
            <label>Network stats:</label>
            <div>
                <div>In: {{ .GlobalConnStats.TotalIn }} ({{ .GlobalConnStats.RateIn }} / s)</div>
                <div>Out: {{ .GlobalConnStats.TotalOut }} ({{ .GlobalConnStats.RateOut }} / s)</div>
            </div>
            <br />

            <label>Peers ({{ .PeersConnected }} connected)</label>
            <ul>
                {{ range .Peers }}
                    <li>
                        <div>
                            {{ .PrettyName }} ({{ .Name }})
                            <form method="post" action="/remove-peer">
                                <input type="hidden" name="peerid" value="{{ .Name }}" />
                                <input type="submit" value="Disconnect" />
                            </form>
                        </div>

                        <ul>
                            {{ range .Addrs }}
                                <li>
                                    {{ . }}
                                </li>
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

            <div><label>Untrack repo</label></div>
            <form action="/untrack-repo" method="post">
                <div>Repo path: <input type="text" name="repoPath" /></div>
                <div><input type="submit" value="Untrack" /></div>
            </form>

            <br />

            <label>Local repos:</label>
            <ul class="local-repos">
                {{ range .LocalRepos }}
                    <li>
                        <div>{{ .RepoID }} ({{ .Path }}) <span class="toggle-refs">[toggle refs]</span></div>
                        <table class="hidden">
                            <thead>
                                <td>Ref</td>
                                <td>Local</td>
                                <td>Remote</td>
                            </thead>
                            <tbody>
                            {{ range .Refs }}
                                <tr>
                                    <td>{{ .RefName }}</td>
                                    <td>{{ .LocalCommit }}</td>
                                    <td>{{ .RemoteCommit }}</td>
                                </tr>
                            {{ end }}
                            </tbody>
                        </table>
                    </li>
                {{ end }}
            </ul>
        </section>

        <section class="section-environment">
            <label>Environment</label>
            <ul>
                {{ range .Env }}
                    <li>{{ .Name }} <span class="equals">=</span> <span class="value">{{ .Value }}</span></li>
                {{ end }}
            </ul>
        </section>

        <section class="section-logs">
            <div>Debug <input type="checkbox" data-level="debug"   value="on" checked /></div>
            <div>Info  <input type="checkbox" data-level="info"    value="on" checked /></div>
            <div>Warn  <input type="checkbox" data-level="warning" value="on" checked /></div>
            <div>Error <input type="checkbox" data-level="error"   value="on" checked /></div>
            <label>Logs:</label>
            <ul></ul>
        </section>

        <script>
            var logs = [
                {{ range .Logs }}
                    { level: {{ .Level }}, message: "{{ .Message }}" },
                {{ end }}
            ]

            function attachListeners() {
                var checkboxes = document.querySelectorAll('section.section-logs input[type=checkbox]')
                for (var i = 0; i < checkboxes.length; i++) {
                    checkboxes[i].addEventListener('click', updateLogs)
                }

                var refToggles = document.querySelectorAll('.local-repos .toggle-refs')
                for (var i = 0; i < refToggles.length; i++) {
                    refToggles[i].addEventListener('click', toggleRefVisibility)
                }
            }

            function toggleRefVisibility(event) {
                console.log('event', event)
                var table = event.target.parentElement.parentElement.querySelector('table')
                table.classList.toggle('hidden')
            }

            function getFilters() {
                var checkboxes = document.querySelectorAll('section.section-logs input[type=checkbox]')
                var filters = {
                    debug: true,
                    info: true,
                    warn: true,
                    error: true,
                }
                for (var i = 0; i < checkboxes.length; i++) {
                    filters[ checkboxes[i].dataset.level ] = checkboxes[i].checked
                }
                return filters
            }

            function updateLogs() {
                var filters = getFilters()

                var ul = document.querySelector('section.section-logs ul')
                ul.innerHTML = ''

                for (var i = 0; i < logs.length; i++) {
                    if (filters[ logs[i].level ] === false) {
                        continue
                    }

                    var li = document.createElement('li')
                    li.innerHTML = logs[i].message
                    li.classList.add('log')
                    li.classList.add(logs[i].level)
                    ul.appendChild(li)
                }
            }

            updateLogs()
            attachListeners()
        </script>
    </body>
    </html>
`))

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
