package ui

var ActionPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>pal - Action</title>
    <link
      rel="stylesheet"
      href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css"
    />
    <link
      rel="stylesheet"
      href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@48,700,1,200"
    />
    <link
      rel="stylesheet"
      href="https://fonts.googleapis.com/css?family=Sixtyfour"
    />
    <link rel="stylesheet" href="/v1/pal/ui/main.css" />
  </head>
  <body>
    <nav
      class="navbar navbar-expand-lg fixed-top navbar-dark bg-dark"
      aria-label="Main navigation"
    >
      <div class="container-xl">
        <a
          class="navbar-brand fs-3"
          style="font-family: Sixtyfour, sans-serif"
          href="/v1/pal/ui"
          >pal</a
        >
      <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarsExample07XL" aria-controls="navbarsExample07XL" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>
        <div class="collapse navbar-collapse" id="navbarsExample07XL">
          <ul class="navbar-nav ms-auto mb-2 mb-lg-0">
            <li class="nav-item">
              <a
                class="nav-link active d-flex fw-bolder"
                aria-current="page"
                href="/v1/pal/ui"
              >
                <span class="material-symbols-outlined me-2"
                  >rule_settings</span
                >
                Actions
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/notifications">
                <span class="material-symbols-outlined me-2">inbox</span>
                Notifications
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/schedules">
                <span class="material-symbols-outlined me-2">schedule</span>
                Schedules
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/files">
                <span class="material-symbols-outlined me-2">description</span>
                Files
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/db">
                <span class="material-symbols-outlined me-2">database</span>
                DB
              </a>
            </li>
            <li class="nav-item">
              <a class="nav-link d-flex fw-bolder" href="/v1/pal/ui/logout">
                <span class="material-symbols-outlined me-2">logout</span>
                Logout
              </a>
            </li>
          </ul>
        </div>
      </div>
    </nav>
    <main class="container">
      <div class="row">
        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
            {{range $group, $action := .}}
              <h5 class="shadow-sm p-3">{{ $action.Desc }}</h5>
              <div class="table-responsive mt-3">
                <table class="table table-hover table-lg table-borderless mb-0">
                  <thead>
                    <tr class="fs-5">
                      <th>Group</th>
                      <th>Action</th>
                      <th class="text-center">Disabled</th>
                      <th class="text-center">Auth</th>
                      <th class="text-center">Background</th>
                      <th class="text-center">Concurrent</th>
                      <th class="text-center">Output</th>
                      <th>Schedule</th>
                      <th>Response Headers</th>
                    </tr>
                  </thead>
                  <tbody>
                	<tr>
                    	<td>
                          	<div class="d-flex">
                            	<div class="fw-bolder fs-5">{{$group}}</div>
                          	</div>
						</td>
                    	<td class="fs-5"><strong>{{$action.Action}}</strong></td>
                    	<td class="text-center fs-5 text-secondary">
            {{ if $action.Disabled }}
							          <a href="/v1/pal/cond/{{$group}}/{{$action.Action}}?disable=false">
                          <span class="material-symbols-outlined me-2 text-success fs-2">circle</span>
                        </a>  
            {{ else }}
							          <a href="/v1/pal/cond/{{$group}}/{{$action.Action}}?disable=true">
                          <span class="material-symbols-outlined me-2 text-secondary fs-2">circle</span>
                        </a>  
            {{ end }}</td>                         
                    	<td class="text-center fs-5 text-secondary">
						{{ if $action.AuthHeader }}
							<span class="material-symbols-outlined me-2 text-success fs-2">circle</span>
        				{{ else }}
							<span class="material-symbols-outlined me-2 fs-2">circle</span>
        				{{ end }}</td>                      
                    	<td class="text-center fs-5 text-secondary">
						{{ if $action.Background }}
							<span class="material-symbols-outlined me-2 text-success fs-2">circle</span>
        				{{ else }}
							<span class="material-symbols-outlined me-2 fs-2">circle</span>
        				{{ end }}</td>
                    	<td class="text-center fs-5 text-secondary">
						{{ if $action.Concurrent }}
							<span class="material-symbols-outlined me-2 text-success fs-2">circle</span>
        				{{ else }}
							<span class="material-symbols-outlined me-2 fs-2">circle</span>
        				{{ end }}</td>						 
                    	<td class="text-center fs-5 text-secondary">
						{{ if $action.Output }}
							<span class="material-symbols-outlined me-2 text-success fs-2">circle</span>
        				{{ else }}
							<span class="material-symbols-outlined me-2 fs-2">circle</span>
        				{{ end }}</td>
                      <td class="fs-5">
                          <a href="/v1/pal/ui/schedules">{{$action.Schedule}}</a>
                      </td>
                    	<td>
                        {{range $header := $action.ResponseHeaders}}
                            <p>{{$header.Header}}: {{$header.Value}}</p>
                        {{end}}
                    	</td>
                	</tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <div class="card shadow-lg">
                <div class="card-body">
                    <div class="mb-3">
                      <label for="inputInput"><strong>Enter Input</strong></label>
          						{{ if $action.InputValidate }}
                      (validations: {{ $action.InputValidate }})
                      {{ end }}
                      <br>
                      <input
                        type="text"
                        class="form-control"
                        id="inputInput"
                        placeholder="INPUT"
                      />
                    </div>
					<button onClick="sendData()" class="btn btn-primary">
                      <span id="runIcon" class="material-symbols-outlined align-bottom"
                        >rule_settings</span
                      >
                      <strong>Run Now</strong>
                    </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      {{end}}

        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <h5 class="card-title">Output <span id="outputStatus" class="material-symbols-outlined align-bottom fs-2"></span></h5>
              <div class="card">
                <div class="card-body bg-dark text-white">
				<pre id="outputPre" class="card-text">
				
				</pre>
                </div>
              </div>
{{range $group, $action := .}}
{{if $action.LastOutput}}
<button class="btn btn-primary"><a href="/v1/pal/ui/action/{{$group}}/{{$action.Action}}/run?last_output=true" target="_blank" class="text-white">
                          <span class="material-symbols-outlined align-bottom">
                            data_object
                          </span>
                          <strong>Last Output</strong>
                        </a></button>
{{end}}
{{end}}
            </div>
          </div>
        </div>

        <div class="col-12 col-lg-12">
          <div class="card">
            <div class="card-body">
              <h5 class="card-title">Cmd</h5>
              <div class="card">
                <div class="card-body bg-dark text-white">
				<pre class="card-text">
{{range $group, $action := .}}
{{$action.Cmd}}
{{end}}
</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
    <script src="/v1/pal/ui/main.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"></script>
  </body>
</html>`
